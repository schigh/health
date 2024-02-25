// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: schigh/health/v1/check.proto

package v1

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Check struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name             string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Healthy          bool                   `protobuf:"varint,2,opt,name=healthy,proto3" json:"healthy,omitempty"`
	AffectsReadiness bool                   `protobuf:"varint,3,opt,name=affects_readiness,json=affectsReadiness,proto3" json:"affects_readiness,omitempty"`
	AffectsLiveness  bool                   `protobuf:"varint,4,opt,name=affects_liveness,json=affectsLiveness,proto3" json:"affects_liveness,omitempty"`
	Error            *structpb.Struct       `protobuf:"bytes,5,opt,name=error,proto3" json:"error,omitempty"`
	ErrorSince       *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=error_since,json=errorSince,proto3" json:"error_since,omitempty"`
}

func (x *Check) Reset() {
	*x = Check{}
	if protoimpl.UnsafeEnabled {
		mi := &file_schigh_health_v1_check_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Check) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Check) ProtoMessage() {}

func (x *Check) ProtoReflect() protoreflect.Message {
	mi := &file_schigh_health_v1_check_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Check.ProtoReflect.Descriptor instead.
func (*Check) Descriptor() ([]byte, []int) {
	return file_schigh_health_v1_check_proto_rawDescGZIP(), []int{0}
}

func (x *Check) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Check) GetHealthy() bool {
	if x != nil {
		return x.Healthy
	}
	return false
}

func (x *Check) GetAffectsReadiness() bool {
	if x != nil {
		return x.AffectsReadiness
	}
	return false
}

func (x *Check) GetAffectsLiveness() bool {
	if x != nil {
		return x.AffectsLiveness
	}
	return false
}

func (x *Check) GetError() *structpb.Struct {
	if x != nil {
		return x.Error
	}
	return nil
}

func (x *Check) GetErrorSince() *timestamppb.Timestamp {
	if x != nil {
		return x.ErrorSince
	}
	return nil
}

var File_schigh_health_v1_check_proto protoreflect.FileDescriptor

var file_schigh_health_v1_check_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x73, 0x63, 0x68, 0x69, 0x67, 0x68, 0x2f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x2f,
	0x76, 0x31, 0x2f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x10,
	0x73, 0x63, 0x68, 0x69, 0x67, 0x68, 0x2e, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x2e, 0x76, 0x31,
	0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xf9, 0x01, 0x0a, 0x05, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a,
	0x07, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07,
	0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x79, 0x12, 0x2b, 0x0a, 0x11, 0x61, 0x66, 0x66, 0x65, 0x63,
	0x74, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64, 0x69, 0x6e, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x10, 0x61, 0x66, 0x66, 0x65, 0x63, 0x74, 0x73, 0x52, 0x65, 0x61, 0x64, 0x69,
	0x6e, 0x65, 0x73, 0x73, 0x12, 0x29, 0x0a, 0x10, 0x61, 0x66, 0x66, 0x65, 0x63, 0x74, 0x73, 0x5f,
	0x6c, 0x69, 0x76, 0x65, 0x6e, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0f,
	0x61, 0x66, 0x66, 0x65, 0x63, 0x74, 0x73, 0x4c, 0x69, 0x76, 0x65, 0x6e, 0x65, 0x73, 0x73, 0x12,
	0x2d, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x3b,
	0x0a, 0x0b, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f, 0x73, 0x69, 0x6e, 0x63, 0x65, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52,
	0x0a, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x53, 0x69, 0x6e, 0x63, 0x65, 0x42, 0x1d, 0x5a, 0x1b, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x63, 0x68, 0x69, 0x67, 0x68,
	0x2f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_schigh_health_v1_check_proto_rawDescOnce sync.Once
	file_schigh_health_v1_check_proto_rawDescData = file_schigh_health_v1_check_proto_rawDesc
)

func file_schigh_health_v1_check_proto_rawDescGZIP() []byte {
	file_schigh_health_v1_check_proto_rawDescOnce.Do(func() {
		file_schigh_health_v1_check_proto_rawDescData = protoimpl.X.CompressGZIP(file_schigh_health_v1_check_proto_rawDescData)
	})
	return file_schigh_health_v1_check_proto_rawDescData
}

var file_schigh_health_v1_check_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_schigh_health_v1_check_proto_goTypes = []interface{}{
	(*Check)(nil),                 // 0: schigh.health.v1.Check
	(*structpb.Struct)(nil),       // 1: google.protobuf.Struct
	(*timestamppb.Timestamp)(nil), // 2: google.protobuf.Timestamp
}
var file_schigh_health_v1_check_proto_depIdxs = []int32{
	1, // 0: schigh.health.v1.Check.error:type_name -> google.protobuf.Struct
	2, // 1: schigh.health.v1.Check.error_since:type_name -> google.protobuf.Timestamp
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_schigh_health_v1_check_proto_init() }
func file_schigh_health_v1_check_proto_init() {
	if File_schigh_health_v1_check_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_schigh_health_v1_check_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Check); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_schigh_health_v1_check_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_schigh_health_v1_check_proto_goTypes,
		DependencyIndexes: file_schigh_health_v1_check_proto_depIdxs,
		MessageInfos:      file_schigh_health_v1_check_proto_msgTypes,
	}.Build()
	File_schigh_health_v1_check_proto = out.File
	file_schigh_health_v1_check_proto_rawDesc = nil
	file_schigh_health_v1_check_proto_goTypes = nil
	file_schigh_health_v1_check_proto_depIdxs = nil
}
