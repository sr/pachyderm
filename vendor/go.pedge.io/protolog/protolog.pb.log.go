// Code generated by protoc-gen-protolog
// source: protolog.proto
// DO NOT EDIT!

package protolog


func init() {
	Register("protolog.Fields", func() Message { return &Fields{} })
	Register("protolog.Event", func() Message { return &Event{} })
	Register("protolog.WriterOutput", func() Message { return &WriterOutput{} })
	Register("protolog.Entry", func() Message { return &Entry{} })
	Register("protolog.Entry.Message", func() Message { return &Entry_Message{} })
}

func (m *Fields) ProtologName() string {
	return "protolog.Fields"
}
func (m *Event) ProtologName() string {
	return "protolog.Event"
}
func (m *WriterOutput) ProtologName() string {
	return "protolog.WriterOutput"
}
func (m *Entry) ProtologName() string {
	return "protolog.Entry"
}
func (m *Entry_Message) ProtologName() string {
	return "protolog.Entry.Message"
}
