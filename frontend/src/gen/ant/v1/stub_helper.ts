// Shared helper for service stubs — returns a GenService that creates
// real client methods. Calls reach the backend, which returns
// CodeUnimplemented (handled silently by the transport error interceptor).
// @ts-nocheck
import type { GenService } from "@bufbuild/protobuf/codegenv2";

export function createStubService(typeName: string): GenService<any> {
  return {
    typeName,
    methods: [],
    method: {},
  } as any;
}
