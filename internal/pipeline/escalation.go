package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/session"
	"github.com/mmariani/ground-control/internal/types"
)

// EscalationReason describes why a task was escalated.
type EscalationReason string

const (
	EscalationMaxIterations    EscalationReason = "max_iterations"
	EscalationReviewerDecision EscalationReason = "reviewer_decision"
	EscalationTesterDecision   EscalationReason = "tester_decision"
	EscalationSanityFailed     EscalationReason = "sanity_failed"
	EscalationUnknown          EscalationReason = "unknown"
)

// Escalation contains all information about an escalated task.
type Escalation struct {
	TaskID         string
	SessionID      string
	Reason         EscalationReason
	Stage          string
	Iteration      int
	Summary        string
	Feedback       string
	AttemptHistory []AttemptRecord
	CreatedAt      time.Time
}

// AttemptRecord documents a single attempt in the feedback loop.
type AttemptRecord struct {
	Iteration   int
	Stage       string
	Status      string
	Feedback    string
	OutputFiles []string
	Timestamp   time.Time
}

// EscalationHandler manages task escalation to human review.
type EscalationHandler struct {
	store   *data.Store
	sessMgr *session.Manager
	dataDir string
}

// NewEscalationHandler creates a new escalation handler.
func NewEscalationHandler(store *data.Store, sessMgr *session.Manager, dataDir string) *EscalationHandler {
	return &EscalationHandler{
		store:   store,
		sessMgr: sessMgr,
		dataDir: dataDir,
	}
}

// Escalate marks a task as escalated and prepares it for human review.
func (h *EscalationHandler) Escalate(task *types.Task, sess *session.Session, reason EscalationReason, feedback string, history []AttemptRecord) (*Escalation, error) {
	escalation := &Escalation{
		TaskID:         task.ID,
		SessionID:      sess.ID,
		Reason:         reason,
		Stage:          sess.CurrentStage,
		Iteration:      sess.CurrentIteration,
		Feedback:       feedback,
		AttemptHistory: history,
		CreatedAt:      time.Now(),
	}

	// Build summary
	escalation.Summary = h.buildEscalationSummary(task, escalation)

	// Save escalation details to file
	if err := h.saveEscalationDetails(escalation); err != nil {
		return nil, fmt.Errorf("saving escalation details: %w", err)
	}

	// Update task state
	if err := h.updateTaskState(task, escalation); err != nil {
		return nil, fmt.Errorf("updating task state: %w", err)
	}

	// Update session
	sess.EscalateTask(task.ID)
	sess.UpdateStatus(session.StatusPaused)
	if err := h.sessMgr.Save(sess); err != nil {
		return nil, fmt.Errorf("saving session: %w", err)
	}

	return escalation, nil
}

func (h *EscalationHandler) buildEscalationSummary(task *types.Task, esc *Escalation) string {
	var summary string

	summary += fmt.Sprintf("# Escalation: %s\n\n", task.Title)
	summary += fmt.Sprintf("**Task ID**: %s\n", task.ID)
	summary += fmt.Sprintf("**Session ID**: %s\n", esc.SessionID)
	summary += fmt.Sprintf("**Stage**: %s (iteration %d)\n", esc.Stage, esc.Iteration)
	summary += fmt.Sprintf("**Reason**: %s\n", esc.Reason)
	summary += fmt.Sprintf("**Time**: %s\n\n", esc.CreatedAt.Format(time.RFC3339))

	summary += "## Task Description\n\n"
	summary += task.Description + "\n\n"

	if len(task.Context.Requirements) > 0 {
		summary += "## Requirements\n\n"
		for _, req := range task.Context.Requirements {
			summary += fmt.Sprintf("- %s\n", req)
		}
		summary += "\n"
	}

	summary += "## Attempt History\n\n"
	for _, attempt := range esc.AttemptHistory {
		summary += fmt.Sprintf("### Iteration %d - %s\n", attempt.Iteration, attempt.Stage)
		summary += fmt.Sprintf("**Status**: %s\n", attempt.Status)
		summary += fmt.Sprintf("**Time**: %s\n", attempt.Timestamp.Format("15:04:05"))
		if attempt.Feedback != "" {
			summary += fmt.Sprintf("**Feedback**: %s\n", attempt.Feedback)
		}
		if len(attempt.OutputFiles) > 0 {
			summary += "**Files**:\n"
			for _, f := range attempt.OutputFiles {
				summary += fmt.Sprintf("  - %s\n", f)
			}
		}
		summary += "\n"
	}

	if esc.Feedback != "" {
		summary += "## Final Feedback\n\n"
		summary += esc.Feedback + "\n\n"
	}

	summary += "## Human Action Required\n\n"
	summary += "Please review the above and choose one of the following:\n\n"
	summary += "1. **Provide Guidance**: Add context/clarification and resume with `gc orc --continue`\n"
	summary += "2. **Take Over**: Complete the task manually\n"
	summary += "3. **Descope**: Reduce the task requirements and retry\n"
	summary += "4. **Abandon**: Mark the task as not needed\n"

	return summary
}

func (h *EscalationHandler) saveEscalationDetails(esc *Escalation) error {
	// Create escalations directory
	escDir := filepath.Join(h.dataDir, "escalations")
	if err := os.MkdirAll(escDir, 0755); err != nil {
		return err
	}

	// Save escalation summary
	filename := fmt.Sprintf("%s_%s.md", esc.TaskID, esc.CreatedAt.Format("20060102_150405"))
	path := filepath.Join(escDir, filename)

	return os.WriteFile(path, []byte(esc.Summary), 0644)
}

func (h *EscalationHandler) updateTaskState(task *types.Task, esc *Escalation) error {
	// Load all tasks
	tasks, err := h.store.LoadTasks()
	if err != nil {
		return err
	}

	// Find and update the task
	for i := range tasks {
		if tasks[i].ID == task.ID {
			tasks[i].State = types.TaskStateWaiting
			tasks[i].UpdatedAt = time.Now()

			// Add escalation note to suggested next steps
			note := fmt.Sprintf("ESCALATED: %s at %s iteration %d. See escalation file for details.",
				esc.Reason, esc.Stage, esc.Iteration)
			tasks[i].SuggestedNext = append(tasks[i].SuggestedNext, note)

			break
		}
	}

	return h.store.SaveTasks(tasks)
}

// GetPendingEscalations returns all tasks currently in escalated state.
func (h *EscalationHandler) GetPendingEscalations() ([]types.Task, error) {
	tasks, err := h.store.LoadTasks()
	if err != nil {
		return nil, err
	}

	var escalated []types.Task
	for _, t := range tasks {
		if t.State == types.TaskStateWaiting {
			// Check if it has escalation note
			for _, note := range t.SuggestedNext {
				if len(note) >= 9 && note[:9] == "ESCALATED" {
					escalated = append(escalated, t)
					break
				}
			}
		}
	}

	return escalated, nil
}

// ResolveEscalation handles human resolution of an escalated task.
func (h *EscalationHandler) ResolveEscalation(taskID string, resolution string, humanInput string) error {
	tasks, err := h.store.LoadTasks()
	if err != nil {
		return err
	}

	for i := range tasks {
		if tasks[i].ID == taskID {
			switch resolution {
			case "retry":
				// Clear escalation and return to created state
				tasks[i].State = types.TaskStateCreated
				tasks[i].SuggestedNext = filterEscalationNotes(tasks[i].SuggestedNext)
				if humanInput != "" {
					// Add human guidance to context
					tasks[i].Context.Background += "\n\nHuman guidance: " + humanInput
				}

			case "complete":
				// Mark as completed manually
				tasks[i].State = types.TaskStateCompleted
				now := time.Now()
				tasks[i].CompletedAt = &now

			case "abandon":
				// Mark as completed (abandoned)
				tasks[i].State = types.TaskStateCompleted
				now := time.Now()
				tasks[i].CompletedAt = &now
				tasks[i].SuggestedNext = append(tasks[i].SuggestedNext, "ABANDONED by human decision")
			}

			tasks[i].UpdatedAt = time.Now()
			break
		}
	}

	return h.store.SaveTasks(tasks)
}

func filterEscalationNotes(notes []string) []string {
	var filtered []string
	for _, note := range notes {
		if len(note) < 9 || note[:9] != "ESCALATED" {
			filtered = append(filtered, note)
		}
	}
	return filtered
}
