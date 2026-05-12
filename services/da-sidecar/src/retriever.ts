import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PROTO_PATH = path.resolve(
  __dirname,
  "../proto/retriever/retriever.proto",
);

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const retrieverProto = (protoDescriptor as any).retriever;

export class RetrieverClient {
  private client: any;

  constructor(rpcUrl: string) {
    this.client = new retrieverProto.Retriever(
      rpcUrl,
      grpc.credentials.createInsecure(),
    );
  }

  retrieveBlob(
    batchHeaderHashHex: string,
    blobIndex: number,
    referenceBlockNumber: number,
    quorumId: number,
  ): Promise<any> {
    return new Promise((resolve, reject) => {
      // Remove '0x' prefix if present and convert hex string to Uint8Array
      const hashStr = batchHeaderHashHex.startsWith("0x")
        ? batchHeaderHashHex.slice(2)
        : batchHeaderHashHex;
      const batchHeaderHashBytes = Buffer.from(hashStr, "hex");

      const request = {
        batch_header_hash: new Uint8Array(batchHeaderHashBytes),
        blob_index: blobIndex,
        reference_block_number: referenceBlockNumber,
        quorum_id: quorumId,
      };

      this.client.RetrieveBlob(request, (error: any, response: any) => {
        if (error) {
          reject(error);
        } else {
          resolve(response);
        }
      });
    });
  }
}
