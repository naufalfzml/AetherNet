import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PROTO_PATH = path.resolve(
  __dirname,
  "../proto/disperser/disperser.proto",
);

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const disperserProto = (protoDescriptor as any).disperser;

export class DisperserClient {
  private client: any;

  constructor(rpcUrl: string) {
    this.client = new disperserProto.Disperser(
      rpcUrl,
      grpc.credentials.createInsecure(),
    );
  }

  disperseBlob(data: Buffer, quorumId: number): Promise<any> {
    return new Promise((resolve, reject) => {
      const request = {
        data: new Uint8Array(data),
        security_params: [
          {
            quorum_id: quorumId,
            adversary_threshold: 21,
            quorum_threshold: 33,
          },
        ],
        target_chunk_num: 0,
      };

      this.client.DisperseBlob(request, (error: any, response: any) => {
        if (error) {
          reject(error);
        } else {
          resolve(response);
        }
      });
    });
  }

  getBlobStatus(requestId: Buffer): Promise<any> {
    return new Promise((resolve, reject) => {
      const request = {
        request_id: new Uint8Array(requestId),
      };

      this.client.GetBlobStatus(request, (error: any, response: any) => {
        if (error) {
          reject(error);
        } else {
          resolve(response);
        }
      });
    });
  }
}
