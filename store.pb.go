// Code generated by protoc-gen-go.
// source: store.proto
// DO NOT EDIT!

/*
Package main is a generated protocol buffer package.

It is generated from these files:
	store.proto

It has these top-level messages:
	StoredPostingsList
	StoredSegment
*/
package main

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type StoredPostingsList struct {
	Term string  `protobuf:"bytes,1,opt,name=term" json:"term,omitempty"`
	Ids  []int32 `protobuf:"varint,2,rep,name=ids" json:"ids,omitempty"`
}

func (m *StoredPostingsList) Reset()         { *m = StoredPostingsList{} }
func (m *StoredPostingsList) String() string { return proto.CompactTextString(m) }
func (*StoredPostingsList) ProtoMessage()    {}

type StoredSegment struct {
	Documents []string              `protobuf:"bytes,1,rep,name=documents" json:"documents,omitempty"`
	Postings  []*StoredPostingsList `protobuf:"bytes,2,rep,name=postings" json:"postings,omitempty"`
}

func (m *StoredSegment) Reset()         { *m = StoredSegment{} }
func (m *StoredSegment) String() string { return proto.CompactTextString(m) }
func (*StoredSegment) ProtoMessage()    {}

func (m *StoredSegment) GetPostings() []*StoredPostingsList {
	if m != nil {
		return m.Postings
	}
	return nil
}

func init() {
	proto.RegisterType((*StoredPostingsList)(nil), "main.StoredPostingsList")
	proto.RegisterType((*StoredSegment)(nil), "main.StoredSegment")
}
