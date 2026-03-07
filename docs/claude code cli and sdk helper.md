Below is a **clear explanation of both**:

1. **Claude Code CLI (headless / non-interactive)**

2. **Claude Agent SDK**

with a focus on **how context is supposed to be passed** in each case.

---

# **Claude Code Context Guide**

## **(CLI Headless \+ Agent SDK)**

This document explains the **intended ways to pass context** when using:

* **Claude Code CLI in headless / non-interactive mode**

* **Claude Agent SDK**

The key design principle across both is:

Claude Code expects most context to come from the **filesystem and project structure**, not from manually constructing huge prompt transcripts.

---

# **1\. Claude Code CLI (Headless / Non-Interactive)**

## **What headless mode is**

Headless mode means running Claude as a **single command that executes once and exits**, instead of opening the interactive TUI.

Typical commands:

claude \-p "summarize this repository"

or

claude \-p "review this diff"

It is designed for:

* CI workflows

* automation scripts

* git hooks

* batch processing

* pipelines

---

# **The Core Idea: Context Comes From the Environment**

When you run Claude headless, context typically comes from **four places**:

1. **Current working directory**

2. **CLAUDE.md project instructions**

3. **stdin (piped input)**

4. **the prompt**

Instead of:

prompt \= gigantic transcript \+ files \+ instructions

Claude Code expects:

filesystem \+ project instructions \+ prompt  
---

# **Context Layer Model (Important)**

Think of Claude Code context as layers:

Prompt  
   immediate task

CLAUDE.md  
   persistent project instructions

Repository Files  
   working context

stdin  
   temporary data (diffs, logs, etc)  
---

# **Passing Context in CLI Headless Mode**

## **1\. Using the Working Directory**

Claude automatically reads files from the **current directory**.

Example:

cd my-project  
claude \-p "explain the architecture"

Claude can inspect:

src/  
README.md  
package.json  
tests/  
---

## **2\. Using `CLAUDE.md` (Project Context)**

`CLAUDE.md` acts as **persistent instructions for the project**.

Example:

CLAUDE.md

Example content:

This repository uses a layered architecture.

Rules:  
\- Use TypeScript  
\- Prefer functional utilities  
\- Avoid adding new dependencies

Claude loads this automatically.

This acts like a **project-level system prompt**.

---

## **3\. Passing Context via stdin**

This is the **most common headless automation pattern**.

Example:

### **Git diff review**

git diff | claude \-p "Review this diff and identify issues"

### **Log analysis**

cat logs.txt | claude \-p "Find errors in these logs"

### **CI test failure analysis**

pytest 2\>&1 | claude \-p "Explain why these tests failed"

This is the **recommended pattern for large temporary context**.

---

## **4\. Prompt**

The prompt should be **task-focused**, not full context.

Good:

Review this diff and identify bugs.

Bad:

Here is my entire codebase:  
\[paste 20k lines\]

Claude Code is designed to **inspect the repository itself**.

---

# **Example Automation Workflow**

Example CI check:

git diff origin/main...HEAD \\  
 | claude \-p "Review this PR and list possible bugs"

Example test failure analysis:

npm test 2\>&1 \\  
 | claude \-p "Explain the root cause of these failures"

Example architecture review:

claude \-p "Explain the architecture of this repository"  
---

# **Session / Conversation Context (Optional)**

Claude CLI can also support **continuing sessions**, but most automation workflows **do not rely on this**.

Instead they use **stateless runs with filesystem context**.

---

# **When to Use CLI Headless**

Best for:

* CI/CD

* PR review bots

* commit hooks

* log analysis

* batch code refactors

* shell pipelines

---

# **2\. Claude Agent SDK**

The **Agent SDK** is the programmatic interface that runs the same Claude Code agent.

Instead of shell commands, you run Claude via code.

Example (Python):

from claude\_agent\_sdk import query  
---

# **Basic SDK Example**

from claude\_agent\_sdk import query, ClaudeAgentOptions  
import asyncio

async def main():  
   options \= ClaudeAgentOptions(  
       cwd=".",  
       allowed\_tools=\["Read", "Edit", "Write", "Bash"\]  
   )

   async for message in query(  
       prompt="Explain this repository",  
       options=options  
   ):  
       print(message)

asyncio.run(main())  
---

# **How Context Is Passed in the SDK**

The SDK passes context through **structured options**.

Important fields:

### **`cwd`**

Project directory Claude should analyze.

Example:

cwd="/repo"

This is equivalent to running the CLI inside that directory.

---

### **`add_dirs`**

Extra directories Claude can read.

Example:

add\_dirs=\[  
   "/shared-docs",  
   "/design-specs"  
\]  
---

### **`system_prompt`**

Adds **behavior instructions**.

Example:

Prefer minimal safe edits.  
Output JSON.  
Follow our Python style guide.  
---

### **`setting_sources`**

Controls whether project settings are loaded.

Example:

setting\_sources=\["project"\]

This enables:

CLAUDE.md  
.claude/settings.json

Important detail:

The SDK **does not load project settings by default**.

You must opt in.

---

### **`prompt`**

The task you want Claude to perform.

Example:

prompt="Refactor this module to remove duplication"  
---

# **SDK Context Model**

SDK context layers:

Prompt  
   task request

system\_prompt  
   behavioral instructions

CLAUDE.md  
   project rules

filesystem  
   repository context  
---

# **Example: Automated Refactor**

options \= ClaudeAgentOptions(  
   cwd="/repo",  
   allowed\_tools=\["Read", "Edit", "Write"\],  
   setting\_sources=\["project"\]  
)

async for msg in query(  
   prompt="Refactor utils.py to remove duplication",  
   options=options  
):  
   print(msg)  
---

# **CLI vs SDK — Key Differences**

| Feature | CLI | SDK |
| ----- | ----- | ----- |
| Best for | shell automation | applications |
| Interface | terminal | Python/JS |
| Context | cwd \+ stdin | cwd \+ options |
| Sessions | possible | easier |
| Tools | built-in | configurable |

---

# **When to Use CLI vs SDK**

Use **CLI** when:

* writing shell automation

* running CI pipelines

* analyzing logs

* reviewing git diffs

Use **SDK** when:

* embedding Claude in apps

* building developer tools

* building AI coding assistants

* creating multi-step workflows

---

# **Practical Mental Model**

Think of Claude Code like this:

Repository \= working memory  
CLAUDE.md \= project brain  
Prompt \= task  
stdin \= temporary data

You generally **should not stuff everything into the prompt**.

Claude Code is designed to **discover context itself**.

