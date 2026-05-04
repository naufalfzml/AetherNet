import { createHash } from "node:crypto";

export type ProofOfInference = {
  modelId: string;
  inputHash: string;
  outputHash: string;
  teeSig: string;
};

export function sha256Hex(value: string | Buffer): string {
  return `0x${createHash("sha256").update(value).digest("hex")}`;
}

export function deterministicStubProof(
  modelId: string,
  input: string,
  output: string,
): ProofOfInference {
  const inputHash = sha256Hex(input);
  const outputHash = sha256Hex(output);
  return {
    modelId,
    inputHash,
    outputHash,
    teeSig: sha256Hex(`${modelId}:${inputHash}:${outputHash}:stub-tee`),
  };
}
