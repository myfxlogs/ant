package strategy

import (
	antv1 "alphaforge/gen/proto/ant/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// paramsProtoToStruct converts proto binary StrategyParams to proto Struct.
func paramsProtoToStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var sp antv1.StrategyParams
	if err := proto.Unmarshal(raw, &sp); err != nil {
		return nil
	}
	m := make(map[string]any, len(sp.GetValues()))
	for k, v := range sp.GetValues() {
		m[k] = v
	}
	s, _ := structpb.NewStruct(m)
	return s
}

// scoreProtoToStruct converts proto binary ScoreComponents to proto Struct.
func scoreProtoToStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var sc antv1.ScoreComponents
	if err := proto.Unmarshal(raw, &sc); err != nil {
		return nil
	}
	m := make(map[string]any, len(sc.GetComponents()))
	for k, v := range sc.GetComponents() {
		m[k] = v
	}
	s, _ := structpb.NewStruct(m)
	return s
}

// spaceProtoToStruct reads proto binary structpb.Struct (stored as BYTEA).
func spaceProtoToStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var ps structpb.Struct
	if err := proto.Unmarshal(raw, &ps); err != nil {
		return nil
	}
	return &ps
}
