"""Minimal pb2 descriptor for StrategyImportService.

Bypasses protobuf compilation — defines just the service descriptor
that connectrpc_server.py needs for endpoint registration.
"""

from google.protobuf import descriptor_pb2
from google.protobuf import descriptor_pool


def _build_service_descriptor() -> descriptor_pb2.ServiceDescriptorProto:
    """Build StrategyImportService descriptor programmatically."""
    pool = descriptor_pool.Default()

    # Check if already registered
    try:
        return pool.FindServiceByName("ant.v1.StrategyImportService")
    except KeyError:
        pass

    file_proto = descriptor_pb2.FileDescriptorProto()
    file_proto.name = "strategy_import.proto"
    file_proto.package = "ant.v1"
    file_proto.syntax = "proto3"

    # -- Message types referenced by the service --
    def _add_msg(name, fields):
        m = file_proto.message_type.add()
        m.name = name
        for fname, ftype, fnum in fields:
            f = m.field.add()
            f.name = fname
            f.number = fnum
            f.label = descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL
            if ftype == "string":
                f.type = descriptor_pb2.FieldDescriptorProto.TYPE_STRING
            elif ftype == "double":
                f.type = descriptor_pb2.FieldDescriptorProto.TYPE_DOUBLE
            elif ftype == "int32":
                f.type = descriptor_pb2.FieldDescriptorProto.TYPE_INT32
            elif ftype == "bool":
                f.type = descriptor_pb2.FieldDescriptorProto.TYPE_BOOL
    # Minimal message definitions (only what's needed for descriptor resolution)
    _add_msg("AnalyzeCodeRequest", [
        ("source_code", "string", 1), ("source_name", "string", 2), ("source_lang", "string", 3)])
    _add_msg("AnalyzeCodeResponse", [
        ("strategy_name", "string", 1), ("coverage_score", "double", 10)])
    _add_msg("GenerateCodeRequest", [
        ("source_code", "string", 1), ("source_name", "string", 2)])
    _add_msg("GenerateCodeResponse", [
        ("python_code", "string", 1), ("compiles", "bool", 3)])
    _add_msg("ImportStrategyRequest", [
        ("source_code", "string", 1), ("source_name", "string", 2)])
    _add_msg("ImportStrategyResponse", [
        ("strategy_id", "string", 1), ("python_code", "string", 3)])

    svc = file_proto.service.add()
    svc.name = "StrategyImportService"

    # AnalyzeCode
    method = svc.method.add()
    method.name = "AnalyzeCode"
    method.input_type = ".ant.v1.AnalyzeCodeRequest"
    method.output_type = ".ant.v1.AnalyzeCodeResponse"

    # GenerateCode
    method = svc.method.add()
    method.name = "GenerateCode"
    method.input_type = ".ant.v1.GenerateCodeRequest"
    method.output_type = ".ant.v1.GenerateCodeResponse"

    # ImportStrategy
    method = svc.method.add()
    method.name = "ImportStrategy"
    method.input_type = ".ant.v1.ImportStrategyRequest"
    method.output_type = ".ant.v1.ImportStrategyResponse"

    # Register in global pool
    serialized = file_proto.SerializeToString()
    pool.AddSerializedFile(serialized)

    # Return the FILE descriptor (connectrpc uses services_by_name on file descriptor)
    return pool.FindFileByName("strategy_import.proto")


DESCRIPTOR = _build_service_descriptor()


# ── Message classes (minimal stubs for ConnectRPC) ──────────────────

from google.protobuf.message import Message

class _StubMessage(Message):
    """Minimal protobuf Message that ConnectRPC can instantiate."""
    def __init__(self, **kwargs):
        for k, v in kwargs.items():
            setattr(self, k, v)

    def ParseFromString(self, _s): pass
    def SerializeToString(self): return b""
    def SerializePartialToString(self): return b""
    def Clear(self): pass
    def IsInitialized(self): return True
    def MergeFrom(self, _other): pass
    def CopyFrom(self, _other): pass
    def ByteSize(self): return 0
    def ListFields(self): return []
    def HasField(self, _name): return False
    def ClearField(self, _name): pass
    def WhichOneof(self, _name): return None
    def HasExtension(self, _ext): return False
    def ClearExtension(self, _ext): pass
    def FindInitializationErrors(self): return []


AnalyzeCodeRequest = _StubMessage
AnalyzeCodeResponse = _StubMessage
GenerateCodeRequest = _StubMessage
GenerateCodeResponse = _StubMessage
ImportStrategyRequest = _StubMessage
ImportStrategyResponse = _StubMessage
