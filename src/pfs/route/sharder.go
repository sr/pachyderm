package route

import (
	"crypto/sha512"
	"encoding/base64"
	"hash/adler32"
	"path"

	"github.com/pachyderm/pachyderm/src/pfs"
)

type sharder struct {
	numShards   uint64
	numReplicas uint64
}

func newSharder(numShards uint64, numReplicas uint64) *sharder {
	return &sharder{numShards, numReplicas}
}

func (s *sharder) NumShards() uint64 {
	return s.numShards
}

func (s *sharder) NumReplicas() uint64 {
	return s.numReplicas
}

func (s *sharder) GetBlock(value []byte) *pfs.Block {
	hash := sha512.Sum512(value)
	return &pfs.Block{
		Hash: base64.URLEncoding.EncodeToString(hash[:]),
	}
}

func (s *sharder) GetShard(file *pfs.File) uint64 {
	return uint64(adler32.Checksum([]byte(path.Clean(file.Path)))) % s.numShards
}

func (s *sharder) GetBlockShard(block *pfs.Block) uint64 {
	return uint64(adler32.Checksum([]byte(block.Hash))) % s.numShards
}
