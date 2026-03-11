package topics

import (
	addressv1 "github.com/bexprt/bexgen-client/pb/address/v1"
	classificationv1 "github.com/bexprt/bexgen-client/pb/classification/v1"
	embeddingv1 "github.com/bexprt/bexgen-client/pb/embedding/v1"
	filev1 "github.com/bexprt/bexgen-client/pb/file/v1"

	"google.golang.org/protobuf/proto"
)

type Topic[T proto.Message] struct {
	Name string
	New  func() T
}

var (
	DocumentUploaded = Topic[*filev1.FileUpload]{
		Name: "document.uploaded",
		New:  func() *filev1.FileUpload { return &filev1.FileUpload{} },
	}

	DocumentOCRCompleted = Topic[*filev1.OcrResult]{
		Name: "document.ocr.completed",
		New:  func() *filev1.OcrResult { return &filev1.OcrResult{} },
	}

	DocumentEmbeddingCreated = Topic[*embeddingv1.EmbeddingResult]{
		Name: "document.embedding.created",
		New:  func() *embeddingv1.EmbeddingResult { return &embeddingv1.EmbeddingResult{} },
	}

	DocumentClassificationCompleted = Topic[*classificationv1.ClassifyResponse]{
		Name: "document.classification.completed",
		New:  func() *classificationv1.ClassifyResponse { return &classificationv1.ClassifyResponse{} },
	}

	DocumentAddressesExtracted = Topic[*addressv1.ExtractFieldsResponse]{
		Name: "document.addresses.extracted",
		New:  func() *addressv1.ExtractFieldsResponse { return &addressv1.ExtractFieldsResponse{} },
	}
)
