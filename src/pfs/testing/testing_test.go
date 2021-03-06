package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pachyderm/pachyderm/src/pfs"
	"github.com/pachyderm/pachyderm/src/pfs/fuse"
	"github.com/pachyderm/pachyderm/src/pfs/pfsutil"
	"github.com/pachyderm/pachyderm/src/pfs/route"
	"github.com/pachyderm/pachyderm/src/pkg/require"
)

const (
	testSize = 200
)

func TestSimple(t *testing.T) {
	t.Parallel()
	RunTest(t, testSimple)
}

func TestFailures(t *testing.T) {
	t.Parallel()
	RunTest(t, testFailures)
}

func TestFuseMount(t *testing.T) {
	t.Skip()
	t.Parallel()
	RunTest(t, testMount)
}

func TestFuseMountBig(t *testing.T) {
	t.Skip()
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	RunTest(t, testMountBig)

}

func BenchmarkFuse(b *testing.B) {
	RunBench(b, benchMount)
}

func testSimple(t *testing.T, apiClient pfs.ApiClient, internalAPIClient pfs.InternalApiClient, cluster Cluster) {
	repositoryName := "testSimpleRepo"

	err := pfsutil.CreateRepo(apiClient, repositoryName)
	require.NoError(t, err)

	scratchCommitInfo, err := pfsutil.InspectCommit(apiClient, repositoryName, "scratch")
	require.NoError(t, err)
	require.NotNil(t, scratchCommitInfo)
	require.Equal(t, "scratch", scratchCommitInfo.Commit.Id)
	require.Equal(t, pfs.CommitType_COMMIT_TYPE_READ, scratchCommitInfo.CommitType)
	require.Nil(t, scratchCommitInfo.ParentCommit)

	commitInfos, err := pfsutil.ListCommit(apiClient, repositoryName)
	require.NoError(t, err)
	require.Equal(t, 1, len(commitInfos))
	require.Equal(t, scratchCommitInfo.Commit, commitInfos[0].Commit)

	commit, err := pfsutil.StartCommit(apiClient, repositoryName, "scratch")
	require.NoError(t, err)
	require.NotNil(t, commit)
	newCommitID := commit.Id

	newCommitInfo, err := pfsutil.InspectCommit(apiClient, repositoryName, newCommitID)
	require.NoError(t, err)
	require.NotNil(t, newCommitInfo)
	require.Equal(t, newCommitID, newCommitInfo.Commit.Id)
	require.Equal(t, pfs.CommitType_COMMIT_TYPE_WRITE, newCommitInfo.CommitType)
	require.Equal(t, "scratch", newCommitInfo.ParentCommit.Id)

	commitInfos, err = pfsutil.ListCommit(apiClient, repositoryName)
	require.NoError(t, err)
	require.Equal(t, 2, len(commitInfos))
	require.Equal(t, newCommitInfo.Commit, commitInfos[0].Commit)
	require.Equal(t, scratchCommitInfo.Commit, commitInfos[1].Commit)

	err = pfsutil.MakeDirectory(apiClient, repositoryName, newCommitID, "a/b")
	require.NoError(t, err)
	err = pfsutil.MakeDirectory(apiClient, repositoryName, newCommitID, "a/c")
	require.NoError(t, err)
	err = pfsutil.MakeDirectory(apiClient, repositoryName, newCommitID, "a/d")
	require.NoError(t, err)

	doWrites(t, apiClient, repositoryName, newCommitID)
	doBlockWrites(t, apiClient, repositoryName, newCommitID)

	err = pfsutil.FinishCommit(apiClient, repositoryName, newCommitID)
	require.NoError(t, err)

	newCommitInfo, err = pfsutil.InspectCommit(apiClient, repositoryName, newCommitID)
	require.NoError(t, err)
	require.NotNil(t, newCommitInfo)
	require.Equal(t, newCommitID, newCommitInfo.Commit.Id)
	require.Equal(t, pfs.CommitType_COMMIT_TYPE_READ, newCommitInfo.CommitType)
	require.Equal(t, "scratch", newCommitInfo.ParentCommit.Id)

	checkWrites(t, apiClient, repositoryName, newCommitID)
	checkBlockWrites(t, apiClient, repositoryName, newCommitID)

	fileInfos, err := pfsutil.ListFile(apiClient, repositoryName, newCommitID, "a/b", 0, 1)
	require.NoError(t, err)
	require.Equal(t, testSize, len(fileInfos))
	fileInfos, err = pfsutil.ListFile(apiClient, repositoryName, newCommitID, "a/c", 0, 1)
	require.NoError(t, err)
	require.Equal(t, testSize, len(fileInfos))

	var fileInfos2 [7][]*pfs.FileInfo
	var wg sync.WaitGroup
	for i := 0; i < 7; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			fileInfos3, iErr := pfsutil.ListFile(apiClient, repositoryName, newCommitID, "a/b", uint64(i), 7)
			require.NoError(t, iErr)
			fileInfos2[i] = fileInfos3
		}()
	}
	wg.Wait()
	count := 0
	for i := 0; i < 7; i++ {
		count += len(fileInfos2[i])
	}
	require.Equal(t, testSize, count)
}

func testFailures(t *testing.T, apiClient pfs.ApiClient, internalAPIClient pfs.InternalApiClient, cluster Cluster) {
	repositoryName := "testFailuresRepo"

	err := pfsutil.CreateRepo(apiClient, repositoryName)
	require.NoError(t, err)

	commit, err := pfsutil.StartCommit(apiClient, repositoryName, "scratch")
	require.NoError(t, err)
	require.NotNil(t, commit)
	newCommitID := commit.Id

	err = pfsutil.MakeDirectory(apiClient, repositoryName, newCommitID, "a/b")
	require.NoError(t, err)
	err = pfsutil.MakeDirectory(apiClient, repositoryName, newCommitID, "a/c")
	require.NoError(t, err)

	doWrites(t, apiClient, repositoryName, newCommitID)

	err = pfsutil.FinishCommit(apiClient, repositoryName, newCommitID)
	require.NoError(t, err)

	checkWrites(t, apiClient, repositoryName, newCommitID)

	for server := 0; server < testNumReplicas; server++ {
		cluster.Kill(server)
	}
	cluster.WaitForAvailability()

	checkWrites(t, apiClient, repositoryName, newCommitID)
}

func testMount(t *testing.T, apiClient pfs.ApiClient, internalAPIClient pfs.InternalApiClient, cluster Cluster) {
	repositoryName := "testMountRepo"

	err := pfsutil.CreateRepo(apiClient, repositoryName)
	require.NoError(t, err)

	directory := "/compile/testMount"
	mounter := fuse.NewMounter(apiClient)
	err = mounter.Mount(repositoryName, directory, "", 0, 1)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(directory, "scratch"))
	require.NoError(t, err)

	commit, err := pfsutil.StartCommit(apiClient, repositoryName, "scratch")
	require.NoError(t, err)
	require.NotNil(t, commit)
	newCommitID := commit.Id

	_, err = os.Stat(filepath.Join(directory, newCommitID))
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(directory, newCommitID, "foo"), []byte("foo"), 0666)
	require.NoError(t, err)

	_, err = pfsutil.PutFile(apiClient, repositoryName, newCommitID, "bar", 0, strings.NewReader("bar"))
	require.NoError(t, err)

	bigValue := make([]byte, 1024*1024)
	for i := 0; i < 1024*1024; i++ {
		bigValue[i] = 'a'
	}

	err = ioutil.WriteFile(filepath.Join(directory, newCommitID, "big1"), bigValue, 0666)
	require.NoError(t, err)

	_, err = pfsutil.PutFile(apiClient, repositoryName, newCommitID, "big2", 0, bytes.NewReader(bigValue))
	require.NoError(t, err)

	err = pfsutil.FinishCommit(apiClient, repositoryName, newCommitID)
	require.NoError(t, err)

	fInfo, err := os.Stat(filepath.Join(directory, newCommitID, "foo"))
	require.NoError(t, err)
	require.Equal(t, int64(3), fInfo.Size())

	data, err := ioutil.ReadFile(filepath.Join(directory, newCommitID, "foo"))
	require.NoError(t, err)
	require.Equal(t, "foo", string(data))

	data, err = ioutil.ReadFile(filepath.Join(directory, newCommitID, "bar"))
	require.NoError(t, err)
	require.Equal(t, "bar", string(data))

	data, err = ioutil.ReadFile(filepath.Join(directory, newCommitID, "big1"))
	require.NoError(t, err)
	require.Equal(t, bigValue, data)

	data, err = ioutil.ReadFile(filepath.Join(directory, newCommitID, "big2"))
	require.NoError(t, err)
	require.Equal(t, bigValue, data)

	err = mounter.Unmount(directory)
	require.NoError(t, err)
	err = mounter.Wait(directory)
	require.NoError(t, err)
}

func testMountBig(t *testing.T, apiClient pfs.ApiClient, internalAPIClient pfs.InternalApiClient, cluster Cluster) {
	repositoryName := "testMountBigRepo"

	err := pfsutil.CreateRepo(apiClient, repositoryName)
	require.NoError(t, err)

	directory := "/compile/testMount"
	mounter := fuse.NewMounter(apiClient)
	err = mounter.Mount(repositoryName, "", directory, 0, 1)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(directory, "scratch"))
	require.NoError(t, err)

	commit, err := pfsutil.StartCommit(apiClient, repositoryName, "scratch")
	require.NoError(t, err)
	require.NotNil(t, commit)
	newCommitID := commit.Id

	bigValue := make([]byte, 1024*1024*300)
	for i := 0; i < 1024*1024*300; i++ {
		bigValue[i] = 'a'
	}

	wg := sync.WaitGroup{}
	for j := 0; j < 5; j++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			err := ioutil.WriteFile(filepath.Join(directory, newCommitID, fmt.Sprintf("big%d", j)), bigValue, 0666)
			require.NoError(t, err)
		}(j)
	}
	wg.Wait()

	err = pfsutil.FinishCommit(apiClient, repositoryName, newCommitID)
	require.NoError(t, err)

	wg = sync.WaitGroup{}
	for j := 0; j < 5; j++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			data, err := ioutil.ReadFile(filepath.Join(directory, newCommitID, fmt.Sprintf("big%d", j)))
			require.NoError(t, err)
			require.Equal(t, bigValue, data)
		}(j)
	}
	wg.Wait()

	err = mounter.Unmount(directory)
	require.NoError(t, err)
	err = mounter.Wait(directory)
	require.NoError(t, err)
}

func benchMount(b *testing.B, apiClient pfs.ApiClient) {
	repositoryName := "benchMountRepo"

	if err := pfsutil.CreateRepo(apiClient, repositoryName); err != nil {
		b.Error(err)
	}

	directory := "/compile/benchMount"
	mounter := fuse.NewMounter(apiClient)
	if err := mounter.Mount(repositoryName, "", directory, 0, 1); err != nil {
		b.Error(err)
	}

	defer func() {
		if err := mounter.Unmount(directory); err != nil {
			b.Error(err)
		}
		if err := mounter.Wait(directory); err != nil {
			b.Error(err)
		}
	}()

	bigValue := make([]byte, 1024*1024)
	for i := 0; i < 1024*1024; i++ {
		bigValue[i] = 'a'
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		commit, err := pfsutil.StartCommit(apiClient, repositoryName, "scratch")
		if err != nil {
			b.Error(err)
		}
		if commit == nil {
			b.Error("nil branch")
		}
		newCommitID := commit.Id
		var wg sync.WaitGroup
		for j := 0; j < 1024; j++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				if err = ioutil.WriteFile(filepath.Join(directory, newCommitID, fmt.Sprintf("big%d", j)), bigValue, 0666); err != nil {
					b.Error(err)
				}
			}(j)
		}
		wg.Wait()
		if err := pfsutil.FinishCommit(apiClient, repositoryName, newCommitID); err != nil {
			b.Error(err)
		}
	}
}

func doWrites(tb testing.TB, apiClient pfs.ApiClient, repositoryName string, commitID string) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < testSize; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, iErr := pfsutil.PutFile(apiClient, repositoryName, commitID,
				fmt.Sprintf("a/b/file%d", i), 0, strings.NewReader(fmt.Sprintf("hello%d", i)))
			require.NoError(tb, iErr)
			_, iErr = pfsutil.PutFile(apiClient, repositoryName, commitID,
				fmt.Sprintf("a/c/file%d", i), 0, strings.NewReader(fmt.Sprintf("hello%d", i)))
			require.NoError(tb, iErr)
		}()
	}
}

func doBlockWrites(tb testing.TB, apiClient pfs.ApiClient, repositoryName string, commitID string) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < testSize; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, iErr := pfsutil.PutBlock(apiClient, repositoryName, commitID,
				fmt.Sprintf("a/d/file%d", i), strings.NewReader(fmt.Sprintf("hello%d", i)))
			require.NoError(tb, iErr)
		}()
	}
}

func checkWrites(tb testing.TB, apiClient pfs.ApiClient, repositoryName string, commitID string) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < testSize; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			buffer := bytes.NewBuffer(nil)
			iErr := pfsutil.GetFile(apiClient, repositoryName, commitID,
				fmt.Sprintf("a/b/file%d", i), 0, math.MaxInt64, buffer)
			require.NoError(tb, iErr)
			require.Equal(tb, fmt.Sprintf("hello%d", i), buffer.String())

			buffer = bytes.NewBuffer(nil)
			iErr = pfsutil.GetFile(apiClient, repositoryName, commitID,
				fmt.Sprintf("a/c/file%d", i), 0, math.MaxInt64, buffer)
			require.NoError(tb, iErr)
			require.Equal(tb, fmt.Sprintf("hello%d", i), buffer.String())

		}()
	}
}

func checkBlockWrites(tb testing.TB, apiClient pfs.ApiClient, repositoryName string, commitID string) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < testSize; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			buffer := bytes.NewBuffer(nil)
			sharder := route.NewSharder(testShardsPerServer*testNumServers, testNumReplicas)
			block := sharder.GetBlock([]byte(fmt.Sprintf("hello%d", i)))
			iErr := pfsutil.GetBlock(apiClient, block.Hash, buffer)
			require.NoError(tb, iErr)

			// buffer = bytes.NewBuffer(nil)
			// require.Equal(tb, fmt.Sprintf("hello%d", i), buffer.String())
			// iErr = pfsutil.GetFile(apiClient, repositoryName, commitID,
			// 	fmt.Sprintf("a/d/file%d", i), 0, math.MaxInt64, buffer)
			// require.NoError(tb, iErr)
			// require.Equal(tb, fmt.Sprintf("hello%d", i), buffer.String())
		}()
	}
}
