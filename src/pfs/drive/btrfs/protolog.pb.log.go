// Code generated by protoc-gen-protolog
// source: pfs/drive/btrfs/protolog.proto
// DO NOT EDIT!

package btrfs

import "go.pedge.io/protolog"

func init() {
	protolog.Register("btrfs.SubvolumeCreate", func() protolog.Message { return &SubvolumeCreate{} })
	protolog.Register("btrfs.SubvolumeDelete", func() protolog.Message { return &SubvolumeDelete{} })
	protolog.Register("btrfs.SubvolumeExists", func() protolog.Message { return &SubvolumeExists{} })
	protolog.Register("btrfs.SubvolumeSnapshot", func() protolog.Message { return &SubvolumeSnapshot{} })
	protolog.Register("btrfs.TransID", func() protolog.Message { return &TransID{} })
	protolog.Register("btrfs.SubvolumeList", func() protolog.Message { return &SubvolumeList{} })
	protolog.Register("btrfs.SubvolumeListLine", func() protolog.Message { return &SubvolumeListLine{} })
	protolog.Register("btrfs.SubvolumeFindNew", func() protolog.Message { return &SubvolumeFindNew{} })
	protolog.Register("btrfs.SubvolumeFindNewLine", func() protolog.Message { return &SubvolumeFindNewLine{} })
	protolog.Register("btrfs.Send", func() protolog.Message { return &Send{} })
	protolog.Register("btrfs.Recv", func() protolog.Message { return &Recv{} })
}

func (m *SubvolumeCreate) ProtologName() string {
	return "btrfs.SubvolumeCreate"
}
func (m *SubvolumeDelete) ProtologName() string {
	return "btrfs.SubvolumeDelete"
}
func (m *SubvolumeExists) ProtologName() string {
	return "btrfs.SubvolumeExists"
}
func (m *SubvolumeSnapshot) ProtologName() string {
	return "btrfs.SubvolumeSnapshot"
}
func (m *TransID) ProtologName() string {
	return "btrfs.TransID"
}
func (m *SubvolumeList) ProtologName() string {
	return "btrfs.SubvolumeList"
}
func (m *SubvolumeListLine) ProtologName() string {
	return "btrfs.SubvolumeListLine"
}
func (m *SubvolumeFindNew) ProtologName() string {
	return "btrfs.SubvolumeFindNew"
}
func (m *SubvolumeFindNewLine) ProtologName() string {
	return "btrfs.SubvolumeFindNewLine"
}
func (m *Send) ProtologName() string {
	return "btrfs.Send"
}
func (m *Recv) ProtologName() string {
	return "btrfs.Recv"
}
