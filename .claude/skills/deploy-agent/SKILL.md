---
name: deploy-agent
description: Walk through the end-to-end workflow for deploying a Claude Code agent on DataGen, from GitHub connection through scheduling. Use when the user asks how to deploy an agent, wants a deployment walkthrough, or needs help setting up their agent pipeline.
---

# End-to-End Agent Deployment Guide

This skill walks you through every step of deploying a Claude Code agent on the DataGen platform using the CLI.

## Prerequisites

- DataGen account with an API key
- A GitHub repository containing `.claude/agents/*.md` files
- The `datagen` CLI installed (`curl -fsSL https://cli.datagen.dev/install.sh | sh`)

## Step 1: Authenticate

```bash
datagen login
```

This saves your API key to your shell profile. Restart your terminal or `source ~/.zshrc` afterward.

Verify authentication works:

```bash
datagen agents list
```

## Step 2: Connect Your GitHub Repository

Install the GitHub App and connect a repository:

```bash
# Start the GitHub App install flow (opens browser)
datagen github connect

# After installing the GitHub App, connect a specific repo
datagen github connect-repo owner/repo-name
```

Verify the connection:

```bash
datagen github repos
```

This automatically discovers all `.claude/agents/*.md`, `.claude/commands/*.md`, and `.claude/skills/*/SKILL.md` files in the repository.

## Step 3: Review Discovered Agents

```bash
# List all discovered agents
datagen agents list

# Show details for a specific agent
datagen agents show <agent-id>
```

Each agent has a UUID. Use `datagen agents list` to find the ID of the agent you want to deploy.

## Step 4: Deploy the Agent

```bash
datagen agents deploy <agent-id>
```

This creates a webhook endpoint. The output includes:
- Webhook URL for triggering the agent
- Webhook token for authentication
- A `curl` example for testing

## Step 5: Configure the Agent

```bash
# Set an entry prompt (overrides the default prompt in the markdown file)
datagen agents config <agent-id> --set-prompt "Process the incoming data and return a summary"

# Attach secrets the agent needs at runtime
datagen agents config <agent-id> --secrets OPENAI_API_KEY,SLACK_TOKEN

# Set PR mode (create_pr, auto_merge, or skip)
datagen agents config <agent-id> --pr-mode create_pr

# Add notification recipients
datagen agents config <agent-id> --add-recipient team@company.com:OWNER
datagen agents config <agent-id> --notify-success true --notify-failure true

# View the full configuration
datagen agents config <agent-id>
```

## Step 6: Test the Agent

```bash
# Run with a test payload
datagen agents run <agent-id> --payload '{"message": "Hello, test run"}'

# Check execution logs
datagen agents logs <agent-id>
```

## Step 7: Set Up a Schedule (Optional)

For recurring execution, create a cron schedule:

```bash
# Run daily at 9am Eastern
datagen agents schedule <agent-id> --cron "0 9 * * *" --timezone "America/New_York" --name "daily-morning"

# Run every Monday at 10am UTC
datagen agents schedule <agent-id> --cron "0 10 * * 1" --name "weekly-monday"

# List schedules
datagen agents schedule <agent-id>

# Pause/resume/delete
datagen agents schedule <agent-id> --pause <schedule-id>
datagen agents schedule <agent-id> --resume <schedule-id>
datagen agents schedule <agent-id> --delete <schedule-id>
```

Common cron expressions:
- `0 9 * * *` -- Every day at 9:00 AM
- `0 9 * * 1-5` -- Weekdays at 9:00 AM
- `0 */6 * * *` -- Every 6 hours
- `0 10 * * 1` -- Every Monday at 10:00 AM
- `0 0 1 * *` -- First day of every month at midnight

## Step 8: Monitor

```bash
# Recent executions
datagen agents logs <agent-id> --limit 20

# Agent details including webhook status
datagen agents show <agent-id>
```

## Quick Reference (Copy-Paste)

Replace `<agent-id>` with your actual agent UUID:

```bash
datagen login
datagen github connect
datagen github connect-repo owner/repo
datagen agents list
datagen agents deploy <agent-id>
datagen agents config <agent-id> --secrets MY_SECRET
datagen agents run <agent-id> --payload '{}'
datagen agents schedule <agent-id> --cron "0 9 * * *" --name "daily"
datagen agents logs <agent-id>
```

## Teardown

```bash
# Remove schedule
datagen agents schedule <agent-id> --delete <schedule-id>

# Undeploy (removes webhook)
datagen agents undeploy <agent-id>
```
