// Code generated by protoc-gen-go.
// source: pps/run/protolog.proto
// DO NOT EDIT!

/*
Package run is a generated protocol buffer package.

It is generated from these files:
	pps/run/protolog.proto

It has these top-level messages:
	AddedPipelineRun
*/
package run

import proto "github.com/golang/protobuf/proto"
import pps "github.com/pachyderm/pachyderm/src/pps"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

type AddedPipelineRun struct {
	PipelineRun *pps.PipelineRun `protobuf:"bytes,1,opt,name=pipeline_run" json:"pipeline_run,omitempty"`
}

func (m *AddedPipelineRun) Reset()         { *m = AddedPipelineRun{} }
func (m *AddedPipelineRun) String() string { return proto.CompactTextString(m) }
func (*AddedPipelineRun) ProtoMessage()    {}

func (m *AddedPipelineRun) GetPipelineRun() *pps.PipelineRun {
	if m != nil {
		return m.PipelineRun
	}
	return nil
}