import { PostDetailShell } from "@/components/post-detail-shell";

export default function Page({ params }: { params: { id: string } }) {
  return <PostDetailShell postID={params.id} />;
}
