# **Claude CLI (Headless) and Claude Agent SDK – Context Guide**

## **Overview**

This document explains the recommended ways to pass context when using:

1\) Claude Code CLI in non‑interactive / headless mode

2\) The Claude Agent SDK

The goal is to clarify how Anthropic expects context to be provided in automation workflows.

## **1\. Claude CLI (Headless / Non‑Interactive)**

Headless mode is typically run with commands like:

claude \-p "your prompt"

claude \-p "your prompt" \--print

This runs Claude once and exits, instead of opening the interactive terminal UI.

Key idea: Claude Code is designed to derive most of its context from the project environment rather than from a giant prompt transcript.

## **How Context Is Normally Passed**

The typical context sources are:

• Current working directory (repository files)

• CLAUDE.md project instructions

• Additional files piped through stdin

• CLI flags controlling tools and permissions

• Existing conversation sessions (optional)

Example:

git diff | claude \-p "Review this diff and suggest improvements"

## **Recommended Context Strategy**

Anthropic’s design encourages structured context layers:

Prompt

    The specific task you want Claude to do now

Repository / Filesystem

    Source code and documentation Claude can read

CLAUDE.md

    Persistent project guidance or conventions

Piped Data

    Temporary context such as diffs, logs, or test results

## **2\. Claude Agent SDK**

The Agent SDK is a programmatic interface that runs the same Claude Code agent behavior from Python or other environments.

Typical usage:

from claude\_agent\_sdk import query, ClaudeAgentOptions

## **Passing Context in the SDK**

Context is passed using structured options rather than large manual transcripts.

Common fields:

cwd

    Project directory Claude should analyze

add\_dirs

    Extra directories Claude can read

system\_prompt

    Global instructions appended to the Claude Code system prompt

setting\_sources

    Whether to load project configuration such as CLAUDE.md

prompt

    The task request itself

## **Key Difference Between CLI and SDK**

CLI

    Designed for shell workflows and pipelines

SDK

    Designed for embedding the agent in applications or services

Both rely heavily on filesystem context instead of prompt stuffing.

## **Practical Mental Model**

Prompt \= What you want done now

CLAUDE.md \= Project memory

Repository \= Working context

CLI flags / SDK options \= Behavior controls