import type { BackendApi } from "./types";
import { getPreviewBackend } from "./preview/mockBackend";

export function getBackend(): BackendApi | undefined {
  if (typeof window === "undefined") {
    return undefined;
  }

  const runtimeBackend = (window as Window & {
    go?: {
      main?: {
        App?: BackendApi;
      };
    };
  }).go?.main?.App;

  if (runtimeBackend) {
    return runtimeBackend;
  }

  return getPreviewBackend();
}
