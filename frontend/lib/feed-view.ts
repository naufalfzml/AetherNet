import type { Agent, Post } from "@/lib/api";

export type DecoratedAgent = Agent & {
  badge: string;
};

export type FeedItem = Post & {
  excerpt: string;
};

export type LiveItem = {
  actor: string;
  action: string;
  target: string;
  age: string;
};

const badgeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

export function getShowcaseAgents(agents: Agent[]): DecoratedAgent[] {
  return agents.map((agent, index) => ({
    ...agent,
    badge: badgeAlphabet[index % badgeAlphabet.length] ?? "A",
  }));
}

export function getTimelineFeed(posts: Post[]): FeedItem[] {
  return posts.map((post) => ({
    ...post,
    excerpt:
      post.text.length > 180
        ? `${post.text.slice(0, 180).trimEnd()}...`
        : post.text,
  }));
}

export function getLiveRail(feed: FeedItem[]): LiveItem[] {
  return feed.slice(0, 5).map((item) => ({
    actor: item.agentId,
    action: "posted",
    target: item.excerpt || "a new dispatch",
    age: formatRelativeTime(item.createdAt),
  }));
}

export function formatRelativeTime(value: string): string {
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) return "Unknown time";

  const diffSeconds = Math.round((timestamp - Date.now()) / 1000);
  const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  const ranges = [
    { unit: "day", seconds: 86_400 },
    { unit: "hour", seconds: 3_600 },
    { unit: "minute", seconds: 60 },
    { unit: "second", seconds: 1 },
  ] as const;

  for (const range of ranges) {
    if (Math.abs(diffSeconds) >= range.seconds || range.unit === "second") {
      return formatter.format(
        Math.trunc(diffSeconds / range.seconds),
        range.unit,
      );
    }
  }

  return "just now";
}
