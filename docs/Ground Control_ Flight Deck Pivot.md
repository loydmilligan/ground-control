# **Design Doc: Ground Control — Flight Deck Pivot**

## **1\. High-Level Vision**

Transition Ground Control from a task-runner to a **Flight Deck**—an orchestration layer that manages high-level state while delegating "Ground Work" to persistent, stateful agent sessions within project directories.

### **Core Concepts**

* **Persistent Sessions**: Use tmux to host long-lived Claude sessions that preserve conversation history and "vibe."  
* **Teleportation**: A CLI/Web feature to instantly attach your terminal to a running agent session.  
* **The Project Sidecar (.gc/)**: A standardized directory injected into every project containing a flight recorder (state.json) and project-specific constraints.  
* **Engagement Altitudes**: Modular workflows that let you "dip your toe" into automation (TDD, Protocol Droid) or stay at "Low Altitude" for manual work.

## **2\. Updated Architecture**

graph TD  
    subgraph "Flight Deck Hub (Global)"  
        WebUI\[Web App / Dashboard\]  
        Daemon\[GC Daemon\]  
        ProtocolDroid\[Protocol Droid Agent\]  
        HubAgents\[Planner / Artifact Gen / Brainstorm\]  
        GlobalDB\[(Global State: tasks, projects, costs)\]  
    end

    subgraph "Project Repo (Local Worktree)"  
        Sidecar\[.gc/state.json\]  
        TmuxSession\[tmux: project-coder\]  
        LocalAgent\[Claude Code CLI\]  
        CliBridge\[mcp2cli / CLI Tools\]  
    end

    Daemon \--\>|Monitors| Sidecar  
    Daemon \--\>|Manages| TmuxSession  
    HubAgents \--\>|Prepares| Sidecar  
    ProtocolDroid \--\>|Quality Gatekeeper| TmuxSession  
    LocalAgent \--\>|Heartbeat/State| Sidecar

## **3\. The gc adopt Workflow**

To homogenize existing projects and bring them into the Flight Deck, we implement an adoption command.

* **Command**: gc adopt \<path\_to\_repo\>  
* **Process**:  
  1. **Survey**: Scans the tech stack (languages, frameworks, test runners).  
  2. **Injection**: Creates the .gc/ directory with state.json and a project-specific CLAUDE.md.  
  3. **Bridge Setup**: Installs mcp2cli patterns to allow the project agent to communicate cost/status back to the Hub.  
  4. **Baseline**: Ingests existing todo.md or issues into Ground Control's global tasks.json.

## **4\. Engagement Altitudes (Optionality)**

Ground Control is non-prescriptive. You choose the level of automation per project:

| Altitude | Level | Description |
| :---- | :---- | :---- |
| **Low** | Interactive | GC manages the tmux session. You do all work manually; GC tracks time/cost. |
| **Mid** | Collaborative | Hub handles Planning, then hands requirements to you in the Repo. |
| **High** | Autonomous | Full TDD Pipeline enabled. Hub generates tests, Repo Coder implements, Droid audits. |

## **5\. Security & Quality (The Protocol Droid Upgrade)**

The Protocol Droid is the upgraded version of our previous monitoring scripts.

### **A. Security: Hashed Approval Protocol**

* **Hashed Auth**: Dangerous tool use (deleting files, net access) requires a password.  
* **Isolation**: Plaintext password never enters the Claude context; only the hash is stored globally.  
* **Handoff**: Requests for approval are piped to the Web UI for mobile/remote decision-making.

### **B. Quality: The TDD Gatekeeper (Optional)**

The Droid enforces the "Red-Green-Refactor" loop:

* **Red-Stage Lock**: The Droid verifies a test fails *before* allowing the agent to modify implementation files.  
* **Cheating Prevention**: Flags attempts by the agent to modify tests just to make them pass existing code.  
* **Loop Kill**: Pauses if an agent runs 5+ commands without making progress toward a "Green" test state.

## **6\. Implementation Inventory**

| Priority | Feature | Description |
| :---- | :---- | :---- |
| **P0** | **tmux Manager** | Internal logic to spawn, list, and attach to sessions. |
| **P0** | **Sidecar Spec** | Flight recorder schema (.gc/state.json). |
| **P1** | **gc adopt** | Logic to bring existing repos into the ecosystem. |
| **P1** | **Web Dashboard** | Dashboard for remote monitoring and "Sliding" approvals. |
| **P1** | **Protocol Droid** | Integration of existing watchdog scripts with the new TDD gatekeeping. |

