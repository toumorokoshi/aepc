// Copyright 2023 Yusuke Fredrick Tsutsumi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package proto

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/parser"
	"github.com/aep-dev/aepc/schema"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// AddResource adds a resource's protos and RPCs to a file and service.
func AddResource(r *parser.ParsedResource, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	resourceMb, err := GeneratedResourceMessage(r)
	if err != nil {
		return fmt.Errorf("unable to generate resource %v: %w", r.Kind, err)
	}
	// set comments for resourceMB
	resourceMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A %v resource.", r.Kind),
	})
	fb.AddMessage(resourceMb)
	if r.Methods != nil {
		if r.Methods.Create != nil {
			err = AddCreate(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Read != nil {
			err = AddGet(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Update != nil {
			err = AddUpdate(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Delete != nil {
			err = AddDelete(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.List != nil {
			err = AddList(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.GlobalList != nil {
			err = AddGlobalList(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}

		if r.Methods.Apply != nil {
			err = AddApply(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GenerateResourceMesssage adds the resource message.
func GeneratedResourceMessage(r *parser.ParsedResource) (*builder.MessageBuilder, error) {
	mb := builder.NewMessage(r.Kind)
	for _, p := range r.GetFieldsSortedByNumber() {
		typ := builder.FieldTypeBool()
		switch p.Type {
		case schema.Type_STRING:
			typ = builder.FieldTypeString()
		case schema.Type_INT32:
			typ = builder.FieldTypeInt32()
		case schema.Type_INT64:
			typ = builder.FieldTypeInt64()
		case schema.Type_BOOLEAN:
			typ = builder.FieldTypeBool()
		case schema.Type_DOUBLE:
			typ = builder.FieldTypeDouble()
		case schema.Type_FLOAT:
			typ = builder.FieldTypeFloat()
		default:
			return nil, fmt.Errorf("proto mapping for type %s not found", p.Type)
		}
		mb.AddField(builder.NewField(p.Name, typ).SetNumber(p.Number).SetComments(
			builder.Comments{
				LeadingComment: fmt.Sprintf("Field for %v.", p.Name),
			},
		))
	}
	mb.SetOptions(
		&descriptorpb.MessageOptions{},
		// annotations.ResourceDescriptor{
		//	"type": sb.GetName() + "/" + r.Kind,
		//},
	)
	// md.GetMessageOptions().ProtoReflect().Set(protoreflect.FieldDescriptor, protoreflect.Value)
	// mb.AddNestedExtension(
	// 	builder.NewExtension("google.api.http", tag int32, typ *builder.FieldType, extendee *builder.MessageBuilder)
	// )
	return mb, nil
}

func AddCreate(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Create" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A Create request for a  %v resource.", r.Kind),
	})
	addParentField(r, mb)
	addIdField(r, mb)
	addResourceField(r, resourceMb, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Create"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Create method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Post{
			// TODO(yft): switch this over to use "id" in the path.
			Post: generateParentHTTPPath(r),
		},
		Body: strings.ToLower(r.Kind),
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PARENT_NAME, strings.ToLower(r.Kind)}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddGet adds a read method for the resource, along with
// any required messages.
func AddGet(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Get" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Get%v method", r.Kind),
	})
	addPathField(r, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Get"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Get method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PATH_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddRead adds a read method for the resource, along with
// any required messages.
func AddUpdate(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Update" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Update%v method", r.Kind),
	})
	addPathField(r, mb)
	addResourceField(r, resourceMb, mb)
	// TODO: find a way to get the actual field mask proto descriptor type, without
	// querying the global registry.
	fieldMaskDescriptor, _ := desc.LoadMessageDescriptorForType(reflect.TypeOf(fieldmaskpb.FieldMask{}))
	mb.AddField(builder.NewField(constants.FIELD_UPDATE_MASK_NAME, builder.FieldTypeImportedMessage(fieldMaskDescriptor)).
		SetNumber(constants.FIELD_UPDATE_MASK_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The update mask for the resource"),
		}))

	fb.AddMessage(mb)
	method := builder.NewMethod("Update"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Update method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Patch{
			Patch: fmt.Sprintf("/{%v.path=%v}", strings.ToLower(r.Kind), generateHTTPPath(r)),
		},
		Body: strings.ToLower(r.Kind),
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{strings.ToLower(r.Kind), constants.FIELD_UPDATE_MASK_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddDelete(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Delete" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Delete%v method", r.Kind),
	})
	addPathField(r, mb)
	fb.AddMessage(mb)
	emptyMd, err := desc.LoadMessageDescriptor("google.protobuf.Empty")
	if err != nil {
		return err
	}
	method := builder.NewMethod("Delete"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeImportedMessage(emptyMd, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Delete method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Delete{
			Delete: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PATH_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddList(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("List" + r.Kind + "Request")
	reqMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the List%v method", r.Kind),
	})
	addParentField(r, reqMb)
	addPageToken(r, reqMb)
	reqMb.AddField(builder.NewField(constants.FIELD_MAX_PAGE_SIZE_NAME, builder.FieldTypeInt32()).
		SetNumber(constants.FIELD_MAX_PAGE_SIZE_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The maximum number of resources to return in a single page."),
		}))
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("List" + r.Kind + "Response")
	respMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Response message for the List%v method", r.Kind),
	})
	addResourcesField(r, resourceMb, respMb)
	addNextPageToken(r, respMb)
	fb.AddMessage(respMb)
	method := builder.NewMethod("List"+r.Kind,
		builder.RpcTypeMessage(reqMb, false),
		builder.RpcTypeMessage(respMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant List method for %v.", r.Plural),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: generateParentHTTPPath(r),
		},
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PARENT_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddGlobalList(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("GlobalList" + r.Kind + "Request")
	reqMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the GlobalList%v method", r.Kind),
	})
	addPathField(r, reqMb)
	addPageToken(r, reqMb)
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("GlobalList" + r.Kind + "Response")
	respMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Response message for the GlobalList%v method", r.Kind),
	})
	addResourcesField(r, resourceMb, respMb)
	addNextPageToken(r, respMb)
	fb.AddMessage(respMb)
	method := builder.NewMethod("GlobalList"+r.Kind,
		builder.RpcTypeMessage(reqMb, false),
		builder.RpcTypeMessage(respMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{path=--/%v}", strings.ToLower(r.Kind)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddApply adds a read method for the resource, along with
// any required messages.
func AddApply(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Apply" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Apply%v method", r.Kind),
	})
	addPathField(r, mb)
	addResourceField(r, resourceMb, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Apply"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Apply method for %v.", r.Plural),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Put{
			Put: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
		// TODO: do a conversion to underscores instead.
		Body: strings.ToLower(r.Kind),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func generateHTTPPath(r *parser.ParsedResource) string {
	elements := []string{strings.ToLower(r.Plural)}
	if len(r.Parents) > 0 {
		// TODO: handle multiple parents
		p := r.Parents[0]
		for p != nil {
			elements = append([]string{strings.ToLower(p.Plural)}, elements...)
			if len(p.Parents) == 0 {
				break
			}
		}
	}
	return fmt.Sprintf("%v/*", strings.Join(elements, "/*/"))
}

func generateParentHTTPPath(r *parser.ParsedResource) string {
	parentPath := ""
	if len(r.Parents) > 0 {
		parentPath = generateHTTPPath(r.Parents[0])
		// parentPath = fmt.Sprintf("{parent=%v/}", generateHTTPPath(r.Parents[0]))
	}
	return fmt.Sprintf("/{parent=%v%v}", parentPath, strings.ToLower(r.Plural))
}

func addParentField(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	proto.SetExtension(o, annotations.E_ResourceReference, &annotations.ResourceReference{})
	f := builder.
		NewField(constants.FIELD_PARENT_NAME, builder.FieldTypeString()).
		SetNumber(constants.FIELD_PARENT_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("A field for the parent of %v", r.Kind),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addIdField(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_ID_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_ID_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An id that uniquely identifies the resource within the collection", r.Kind),
	})
	mb.AddField(f)
}

func addPathField(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	proto.SetExtension(o, annotations.E_ResourceReference, &annotations.ResourceReference{
		Type: r.Type,
	})
	f := builder.NewField(constants.FIELD_PATH_NAME, builder.FieldTypeString()).
		SetNumber(constants.FIELD_PATH_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The globally unique identifier for the resource"),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addResourceField(r *parser.ParsedResource, resourceMb, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	f := builder.NewField(strings.ToLower(r.Kind), builder.FieldTypeMessage(resourceMb)).
		SetNumber(constants.FIELD_RESOURCE_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The resource to perform the operation on."),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addResourcesField(r *parser.ParsedResource, resourceMb, mb *builder.MessageBuilder) {
	f := builder.NewField("results", builder.FieldTypeMessage(resourceMb)).SetNumber(constants.FIELD_RESOURCES_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A list of %v", r.Plural),
	}).SetRepeated()
	mb.AddField(f)
}

func addPageToken(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_PAGE_TOKEN_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_PAGE_TOKEN_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("The page token indicating the starting point of the page"),
	})
	mb.AddField(f)
}

func addNextPageToken(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_NEXT_PAGE_TOKEN_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_NEXT_PAGE_TOKEN_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("The page token indicating the ending point of this response."),
	})
	mb.AddField(f)
}
