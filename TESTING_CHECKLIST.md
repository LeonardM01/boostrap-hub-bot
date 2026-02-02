# Points and Leaderboard Feature - Testing Checklist

This document provides a comprehensive manual testing checklist for the newly implemented points and leaderboard feature.

## Prerequisites

Before testing, ensure you have:
1. Set up the `.env` file with required environment variables:
   - `DISCORD_BOT_TOKEN`
   - `DISCORD_APPLICATION_ID`
   - `DISCORD_GUILD_ID` (optional, for guild-specific registration)
   - `DISCORD_REMINDER_CHANNEL_ID` (optional, for automated reminders)
   - `OPENAI_API_KEY` (optional, for AI-powered point calculation)
   - `DATABASE_PATH` (optional, defaults to `data/bootstrap_hub.db`)

2. Run `make register` to register the new commands with Discord
3. Invite the bot to your test server with proper permissions
4. Start the bot with `make run`

## Test Cases

### Phase 1: Point Calculation (Adding Tasks)

#### Test 1.1: Add task with OpenAI API key configured
- **Command**: `/focus add goal:Launch landing page`
- **Expected**:
  - Task is added successfully
  - Points between 1-10 are displayed (calculated by OpenAI)
  - Embed shows: "**#1:** Launch landing page\n**Points:** X/10"

#### Test 1.2: Add task without OpenAI API key
- **Setup**: Remove or comment out `OPENAI_API_KEY` from `.env`, restart bot
- **Command**: `/focus add goal:Write blog post`
- **Expected**:
  - Task is added successfully
  - Default points (5) are assigned
  - Embed shows: "**#1:** Write blog post\n**Points:** 5/10"

#### Test 1.3: Add multiple tasks with varying complexity
- **Commands**:
  - `/focus add goal:Fix typo in documentation` (expect low points ~1-3)
  - `/focus add goal:Build complete payment integration` (expect high points ~8-10)
  - `/focus add goal:Research competitor pricing` (expect medium points ~4-6)
- **Expected**: Points reflect relative task difficulty

### Phase 2: Earning Points (Completing Tasks)

#### Test 2.1: Complete a task and earn points
- **Setup**: Add a task first
- **Command**: `/focus complete number:1`
- **Expected**:
  - Task is marked as completed
  - Embed shows: "+X points earned!" (where X is the task's point value)
  - Points are added to user's total

#### Test 2.2: Complete multiple tasks
- **Setup**: Add 3 tasks
- **Commands**:
  - `/focus complete number:1`
  - `/focus complete number:2`
  - `/focus complete number:3`
- **Expected**: Points accumulate with each completion

#### Test 2.3: Try to complete already completed task
- **Command**: `/focus complete number:1` (on an already completed task)
- **Expected**: Error message: "task #1 is already completed"

### Phase 3: All-Time Leaderboard

#### Test 3.1: View empty leaderboard
- **Setup**: Fresh database with no completed tasks
- **Command**: `/leaderboard alltime`
- **Expected**:
  - Message: "No one has earned points yet!"
  - Suggestion to start a Focus Period

#### Test 3.2: View leaderboard with entries
- **Setup**: Multiple users with completed tasks
- **Command**: `/leaderboard alltime`
- **Expected**:
  - Top 10 users ranked by total points (descending)
  - Medals for top 3: 🥇 🥈 🥉
  - Format: "🥇 **Username** - X points (Y tasks)"
  - Users with 0 points are excluded

#### Test 3.3: Verify ranking order
- **Setup**:
  - User A completes tasks worth 30 points
  - User B completes tasks worth 50 points
  - User C completes tasks worth 20 points
- **Command**: `/leaderboard alltime`
- **Expected Order**:
  1. User B (50 points)
  2. User A (30 points)
  3. User C (20 points)

### Phase 4: Sprint Leaderboard

#### Test 4.1: View current sprint leaderboard
- **Setup**: Active focus period with completed tasks
- **Command**: `/leaderboard sprint`
- **Expected**:
  - Shows only points earned in current active focus periods
  - Ranked by sprint points (descending)
  - Medals for top 3

#### Test 4.2: Sprint vs All-Time comparison
- **Setup**:
  - User A has 100 all-time points, 10 current sprint points
  - User B has 50 all-time points, 20 current sprint points
- **Commands**:
  - `/leaderboard alltime` - expect User A first
  - `/leaderboard sprint` - expect User B first
- **Expected**: Rankings differ based on context

#### Test 4.3: Sprint leaderboard after period ends
- **Setup**: Wait for focus period to end (or manually adjust database dates)
- **Command**: `/leaderboard sprint`
- **Expected**: Empty or shows only users in new active periods

### Phase 5: Admin Configuration

#### Test 5.1: Set leaderboard channel (admin only)
- **Setup**: User with Administrator permission
- **Command**: `/config leaderboard-channel channel:#leaderboard`
- **Expected**:
  - Success message: "Leaderboard channel set to #leaderboard"
  - Channel is saved to database

#### Test 5.2: Set leaderboard channel (non-admin)
- **Setup**: User without Administrator permission
- **Command**: `/config leaderboard-channel channel:#leaderboard`
- **Expected**: Discord shows error (command not visible or permission denied)

#### Test 5.3: Verify channel configuration persists
- **Setup**: Set leaderboard channel, restart bot
- **Expected**: Configuration persists across bot restarts

### Phase 6: Automated Leaderboard Posting

#### Test 6.1: Leaderboard posts when period ends
- **Setup**:
  - Configure leaderboard channel with `/config`
  - Wait for a focus period to end (or manually adjust database)
  - Ensure scheduler runs (9 AM daily check)
- **Expected**:
  - Leaderboard automatically posts to configured channel
  - Shows top performers from completed sprint
  - Format: "🏆 Sprint Leaderboard - Focus Period Completed!"

#### Test 6.2: No duplicate leaderboard posts
- **Setup**: Period already ended with leaderboard posted
- **Expected**: Leaderboard is not posted again

#### Test 6.3: Multiple users in same period
- **Setup**: 3 users all end focus period on same day
- **Expected**: Single leaderboard post showing all users ranked

### Phase 7: Help Command Updates

#### Test 7.1: View updated help command
- **Command**: `/help`
- **Expected**:
  - Shows new leaderboard commands section
  - Mentions AI point calculation
  - Mentions admin configuration command
  - Updated description for `/focus add` and `/focus complete`

### Phase 8: Integration Tests

#### Test 8.1: Complete workflow
1. `/focus start` - Start new focus period
2. `/focus add goal:Build feature X` - Add task (note points)
3. `/focus add goal:Write tests` - Add another task
4. `/focus list` - Verify tasks listed
5. `/focus complete number:1` - Complete first task (earn points)
6. `/leaderboard alltime` - Verify user appears on leaderboard
7. `/leaderboard sprint` - Verify user appears on sprint leaderboard
8. `/focus complete number:2` - Complete second task
9. `/leaderboard alltime` - Verify points increased
- **Expected**: All commands work together seamlessly

#### Test 8.2: Multiple users competing
- **Setup**: 2+ users in same server
- **Scenario**:
  - User 1 completes 5 tasks worth 25 points
  - User 2 completes 3 tasks worth 27 points
  - User 3 completes 1 task worth 10 points
- **Commands**: Both users run `/leaderboard alltime`
- **Expected**: Correct ranking (User 2, User 1, User 3)

#### Test 8.3: Cross-guild isolation
- **Setup**: Bot in 2+ Discord servers
- **Scenario**:
  - User completes tasks in Server A
  - Same user joins Server B
- **Expected**:
  - Leaderboard in Server A shows user's points
  - Leaderboard in Server B shows 0 points (separate tracking)

### Phase 9: Edge Cases

#### Test 9.1: User with no completed tasks
- **Setup**: Start focus period, add tasks, don't complete any
- **Command**: `/leaderboard alltime`
- **Expected**: User does not appear on leaderboard

#### Test 9.2: Task with 0 points
- **Setup**: Manually set task points to 0 in database
- **Command**: `/focus complete number:X`
- **Expected**: Task completes, 0 points earned, user not on leaderboard

#### Test 9.3: Very long task title
- **Command**: `/focus add goal:[300+ character string]`
- **Expected**: Task added, embed displays properly (may truncate)

#### Test 9.4: OpenAI API error handling
- **Setup**: Invalid OpenAI API key or rate limit exceeded
- **Command**: `/focus add goal:Test task`
- **Expected**:
  - Fallback to default points (5)
  - Task is still added successfully
  - Error logged but not shown to user

#### Test 9.5: Database migration
- **Setup**: Run bot with old database (without points fields)
- **Expected**: Auto-migration adds new fields, existing data preserved

## Performance Tests

### Test P1: Leaderboard with many users
- **Setup**: Database with 100+ users
- **Command**: `/leaderboard alltime`
- **Expected**: Response within 3 seconds (Discord interaction limit)

### Test P2: Concurrent task completions
- **Setup**: Multiple users completing tasks simultaneously
- **Expected**: All points correctly added, no race conditions

## Regression Tests

### Test R1: Existing focus period commands still work
- Verify `/focus start`, `/focus list`, `/focus status` work as before
- Verify reminders still function (days 3, 7, 10, 12, 13)

### Test R2: Existing resource commands unaffected
- Verify `/resource` commands still work (not affected by changes)

## Success Criteria

- [ ] All Phase 1-9 tests pass
- [ ] Performance tests meet requirements
- [ ] No regressions in existing functionality
- [ ] Points system feels fair and motivating
- [ ] AI point calculation is reasonably accurate
- [ ] Leaderboards update correctly in real-time
- [ ] Automated leaderboard posts work on schedule
- [ ] Admin configuration persists and functions correctly

## Notes

- Test with OpenAI API key both enabled and disabled
- Test with scheduler both enabled and disabled
- Verify all database migrations run successfully
- Check logs for any errors during operation
- Test with different Discord permission configurations
- Verify embed formatting looks good on mobile and desktop Discord clients
