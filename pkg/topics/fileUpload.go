package topics

import (
	"reflect"

	pbFile "github.com/bexprt/bexgen-client/pb/file/v1"
	pbUser "github.com/bexprt/bexgen-client/pb/users/v1"

	"google.golang.org/protobuf/proto"
)

type Topic[T proto.Message] struct {
	Name string
}

func New[T proto.Message](name string) *Topic[T] {
	t := &Topic[T]{Name: name}

	typeToName[reflect.TypeOf(new(T))] = name
	nameToType[name] = reflect.TypeOf(new(T))

	return t
}

var (
	typeToName = make(map[reflect.Type]string)
	nameToType = make(map[string]reflect.Type)
)

func NameFromType[T proto.Message]() string {
	return typeToName[reflect.TypeOf(new(T))]
}

func TypeFromName(name string) reflect.Type {
	return nameToType[name]
}

func NewAlloc[T proto.Message]() T {
	var zero T
	return reflect.New(reflect.TypeOf(zero).Elem()).Interface().(T)
}

var (
	FileUpload = New[*pbFile.FileUpload]("file.upload")
	User       = New[*pbUser.User]("user")
)
