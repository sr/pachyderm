package testing

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"go.pedge.io/proto/test"

	"github.com/pachyderm/pachyderm/src/pfs"
	"github.com/pachyderm/pachyderm/src/pfs/drive"
	"github.com/pachyderm/pachyderm/src/pfs/drive/btrfs"
	"github.com/pachyderm/pachyderm/src/pfs/route"
	"github.com/pachyderm/pachyderm/src/pfs/server"
	"github.com/pachyderm/pachyderm/src/pkg/discovery"
	"github.com/pachyderm/pachyderm/src/pkg/grpcutil"
	"github.com/pachyderm/pachyderm/src/pkg/require"
	"google.golang.org/grpc"
)

const (
	// TODO(pedge): large numbers of shards takes forever because
	// we are doing tons of btrfs operations on init, is there anything
	// we can do about that?
	testShardsPerServer = 4
	testNumServers      = 4
	testNumReplicas     = 1
)

var (
	counter int32
)

func RunTest(
	t *testing.T,
	f func(*testing.T, pfs.ApiClient, pfs.InternalApiClient, Cluster),
) {
	discoveryClient, err := getEtcdClient()
	require.NoError(t, err)
	var cluster Cluster
	prototest.RunT(
		t,
		testNumServers,
		func(servers map[string]*grpc.Server) {
			cluster = registerFunc(t, discoveryClient, servers)
		},
		func(t *testing.T, clientConns map[string]*grpc.ClientConn) {
			var clientConn *grpc.ClientConn
			for _, c := range clientConns {
				clientConn = c
				break
			}
			f(
				t,
				pfs.NewApiClient(
					clientConn,
				),
				pfs.NewInternalApiClient(
					clientConn,
				),
				cluster,
			)
		},
	)
	cluster.Shutdown()
}

func RunBench(
	b *testing.B,
	f func(*testing.B, pfs.ApiClient),
) {
	discoveryClient, err := getEtcdClient()
	require.NoError(b, err)
	var cluster Cluster
	prototest.RunB(
		b,
		testNumServers,
		func(servers map[string]*grpc.Server) {
			cluster = registerFunc(b, discoveryClient, servers)
		},
		func(b *testing.B, clientConns map[string]*grpc.ClientConn) {
			var clientConn *grpc.ClientConn
			for _, c := range clientConns {
				clientConn = c
				break
			}
			f(
				b,
				pfs.NewApiClient(
					clientConn,
				),
			)
		},
	)
	cluster.Shutdown()
}

type Cluster interface {
	WaitForAvailability()
	Kill(server int)
	Restart(server int)
	Shutdown()
}

type cluster struct {
	addresses       []string
	servers         map[string]server.APIServer
	internalServers map[string]server.InternalAPIServer
	cancels         map[string]chan bool
	cancel          chan bool
	addresser       route.TestAddresser
	sharder         route.Sharder
	tb              testing.TB
}

func (c *cluster) WaitForAvailability() {
	// We use address as the id for servers too
	var ids []string
	for _, address := range c.addresses {
		if _, ok := c.cancels[address]; ok {
			ids = append(ids, address)
		}
	}
	require.NoError(c.tb, c.addresser.WaitForAvailability(ids))
}

func (c *cluster) Kill(server int) {
	close(c.cancels[c.addresses[server]])
	delete(c.cancels, c.addresses[server])
}

func (c *cluster) Restart(server int) {
	address := c.addresses[server]
	c.cancels[address] = make(chan bool)
	go func() {
		require.Equal(c.tb, c.addresser.Register(c.cancels[address], address, address, c.internalServers[address]), route.ErrCancelled)
	}()
}

func (c *cluster) Shutdown() {
	close(c.cancel)
	for _, cancel := range c.cancels {
		close(cancel)
	}
}

func newCluster(tb testing.TB, discoveryClient discovery.Client, servers map[string]*grpc.Server) Cluster {
	sharder := route.NewSharder(
		testShardsPerServer*testNumServers,
		testNumReplicas,
	)
	addresser := route.NewDiscoveryTestAddresser(discoveryClient, sharder, testNamespace())
	cluster := cluster{
		servers:         make(map[string]server.APIServer),
		internalServers: make(map[string]server.InternalAPIServer),
		cancels:         make(map[string]chan bool),
		cancel:          make(chan bool),
		addresser:       addresser,
		sharder:         sharder,
		tb:              tb,
	}
	for address, s := range servers {
		apiServer := server.NewAPIServer(
			cluster.sharder,
			route.NewRouter(
				cluster.addresser,
				grpcutil.NewDialer(
					grpc.WithInsecure(),
				),
				address,
			),
		)
		pfs.RegisterApiServer(s, apiServer)
		internalAPIServer := server.NewInternalAPIServer(
			cluster.sharder,
			route.NewRouter(
				cluster.addresser,
				grpcutil.NewDialer(
					grpc.WithInsecure(),
				),
				address,
			),
			getDriver(tb, address),
		)
		pfs.RegisterInternalApiServer(s, internalAPIServer)
		cluster.addresses = append(cluster.addresses, address)
		cluster.servers[address] = apiServer
		cluster.internalServers[address] = internalAPIServer
		cluster.cancels[address] = make(chan bool)
		go func(address string) {
			require.Equal(tb, cluster.addresser.Register(cluster.cancels[address], address, address, cluster.internalServers[address]), route.ErrCancelled)
		}(address)
	}
	go func() {
		require.Equal(tb, cluster.addresser.AssignRoles(cluster.cancel), route.ErrCancelled)
	}()
	return &cluster
}

func registerFunc(tb testing.TB, discoveryClient discovery.Client, servers map[string]*grpc.Server) Cluster {
	cluster := newCluster(tb, discoveryClient, servers)
	cluster.WaitForAvailability()
	return cluster
}

func getDriver(tb testing.TB, namespace string) drive.Driver {
	driver, err := btrfs.NewDriver(getBtrfsRootDir(tb), namespace)
	require.NoError(tb, err)
	return driver
}

func getBtrfsRootDir(tb testing.TB) string {
	// TODO(pedge)
	rootDir := os.Getenv("PFS_DRIVER_ROOT")
	if rootDir == "" {
		tb.Fatal("PFS_DRIVER_ROOT not set")
	}
	return rootDir
}

func getEtcdClient() (discovery.Client, error) {
	etcdAddress, err := getEtcdAddress()
	if err != nil {
		return nil, err
	}
	return discovery.NewEtcdClient(etcdAddress), nil
}

func getEtcdAddress() (string, error) {
	etcdAddr := os.Getenv("ETCD_PORT_2379_TCP_ADDR")
	if etcdAddr == "" {
		return "", errors.New("ETCD_PORT_2379_TCP_ADDR not set")
	}
	return fmt.Sprintf("http://%s:2379", etcdAddr), nil
}

func testNamespace() string {
	return fmt.Sprintf("test-%d", atomic.AddInt32(&counter, 1))
}
