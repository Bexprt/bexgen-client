package topics

import (
	pbFile "github.com/bexprt/bexgen-client/pb/file/v1"

	"google.golang.org/protobuf/proto"
)

type Topic[T proto.Message] struct {
	Name string
	New  func() T
}

var (
	FileUpload = Topic[*pbFile.FileUpload]{
		Name: "file.upload",
		New:  func() *pbFile.FileUpload { return &pbFile.FileUpload{} },
	}

	OcrResult = Topic[*pbFile.OcrResult]{
		Name: "ocr.result",
		New:  func() *pbFile.OcrResult { return &pbFile.OcrResult{} },
	}

	Embedding = Topic[*pbFile.OcrResult]{
		Name: "ai.embedding",
		New:  func() *pbFile.OcrResult { return &pbFile.OcrResult{} },
	}
)
