package helper

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/apis/audit"
	v1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	decoder       = codecs.UniversalDecoder()
)

func init() {
	_ = v1.AddToScheme(runtimeScheme)
	_ = audit.AddToScheme(runtimeScheme)
}

// DecodeEventList decodes audit event data from bytes to EventList.
func DecodeEventList(data []byte) (*audit.EventList, error) {
	internal, schemaVersion, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	out, ok := internal.(*audit.EventList)
	if !ok {
		return nil, fmt.Errorf("failure to cast to auditlog internal; type: %v", schemaVersion)
	}
	return out, nil
}

// EncodeEventList encodes audit event data from EventList to bytes.
func EncodeEventList(eventList *audit.EventList) ([]byte, error) {
	return runtime.Encode(codecs.LegacyCodec(v1.SchemeGroupVersion), eventList)
}
