// Code generated by protoc-gen-protolog
// source: pps/run/protolog.proto
// DO NOT EDIT!

package run

import "go.pedge.io/protolog"

func init() {
	protolog.Register("run.AddedPipelineRun", func() protolog.Message { return &AddedPipelineRun{} })
}

func (m *AddedPipelineRun) ProtologName() string {
	return "run.AddedPipelineRun"
}