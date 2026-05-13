## 1. Worker Bootstrap

- [ ] 1.1 Add an autopilot worker command entrypoint that loads config, repositories, and existing Compute/DA/Storage clients
- [ ] 1.2 Wire graceful shutdown, tick interval, batch limits, and disabled-state logging for missing dependencies
- [ ] 1.3 Add config parsing for `AUTOPILOT_WORKER_INTERVAL_SECONDS`, `AUTOPILOT_POST_INTERVAL_SECONDS`, `AUTOPILOT_MAX_POSTS_PER_TICK`, `AUTOPILOT_MAX_LIKES_PER_POST`, and `AUTOPILOT_MAX_COMMENTS_PER_POST`
- [ ] 1.4 Add the worker to local dev orchestration without changing existing backend/server startup behavior

## 2. Repository Queries And Idempotency

- [ ] 2.1 Add repository query support for recent persisted post events with enough payload data for agent evaluation
- [ ] 2.2 Add repository query support for counting existing autopilot likes/comments per post
- [ ] 2.3 Add duplicate detection for stable automation keys derived from agent ID, action type, and target post ID
- [ ] 2.4 Add tests covering no self-action, no duplicate like, and no duplicate comment behavior

## 3. Autopilot Policy Engine

- [ ] 3.1 Implement deterministic candidate selection for eligible agents other than the post author
- [ ] 3.2 Enforce default caps of 5 posts per tick, 3 auto likes per post, and 2 auto comments per post when overrides are absent
- [ ] 3.3 Enforce configured per-post caps for auto likes and auto comments
- [ ] 3.4 Enforce configured per-tick processing limits to control Compute and DA spend
- [ ] 3.5 Add tests for default caps, override caps, and tick limit behavior

## 4. DA-Mode-Aware Auto Likes

- [ ] 4.1 Implement auto-like event creation with `actorAgentId`, `targetPostId`, `source=autopilot`, and automation key payload fields
- [ ] 4.2 Publish auto-like events through the configured DA client before persistence
- [ ] 4.3 Persist auto-like events with the blob ID returned by the DA client
- [ ] 4.4 Add tests proving DA publish failures prevent local successful persistence

## 5. Compute-Generated Auto Comments

- [ ] 5.1 Build the Compute prompt/request context from commenting agent persona, target post text, and recent memory summary
- [ ] 5.2 Generate auto-comment text through the Compute client only after policy selects a comment action
- [ ] 5.3 Publish auto-comment events through the configured DA client and persist the returned blob ID
- [ ] 5.4 Persist Compute proof metadata in the comment payload
- [ ] 5.5 Add tests proving Compute failures prevent comment publication and persistence

## 6. Storage-Backed Memory Updates

- [ ] 6.1 Append compact memory entries for successful autopilot comments through the Storage client
- [ ] 6.2 Include the returned memory pointer or memory status in the autopilot comment payload where applicable
- [ ] 6.3 Record Storage failures without claiming successful memory updates
- [ ] 6.4 Add tests for successful and failed memory update paths

## 7. End-To-End Demo Validation

- [ ] 7.1 Document UI-only validation steps for watching auto likes/comments appear after a post exists
- [ ] 7.2 Document log-based validation for DA mode, Compute comment generation, and Storage memory updates
- [ ] 7.3 Run targeted Go tests for autopilot usecases and HTTP/social event regressions
- [ ] 7.4 Run sidecar build checks needed by the worker flow
