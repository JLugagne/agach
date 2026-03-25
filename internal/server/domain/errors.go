package domain

import "errors"

// Error represents a domain error with a code and message
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// ErrorCode returns the domain error code, satisfying controller.CodedError.
func (e *Error) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the domain error message, satisfying controller.CodedError.
func (e *Error) ErrorMessage() string {
	return e.Message
}

// IsDomainError checks if an error is a domain error
func IsDomainError(err error) bool {
	var domainErr *Error
	return errors.As(err, &domainErr)
}

// Common domain errors
var (
	// Project errors
	ErrProjectNotFound = &Error{
		Code:    "PROJECT_NOT_FOUND",
		Message: "project not found",
	}
	ErrProjectAlreadyExists = &Error{
		Code:    "PROJECT_ALREADY_EXISTS",
		Message: "project already exists",
	}
	ErrInvalidProjectData = &Error{
		Code:    "INVALID_PROJECT_DATA",
		Message: "invalid project data",
	}
	ErrProjectNameRequired = &Error{
		Code:    "PROJECT_NAME_REQUIRED",
		Message: "project name is required",
	}

	// Agent errors
	ErrAgentNotFound = &Error{
		Code:    "AGENT_NOT_FOUND",
		Message: "agent not found",
	}
	ErrAgentAlreadyExists = &Error{
		Code:    "AGENT_ALREADY_EXISTS",
		Message: "agent already exists with this slug",
	}
	ErrAgentInUse = &Error{
		Code:    "AGENT_IN_USE",
		Message: "agent is still in use by tasks",
	}
	ErrInvalidAgentData = &Error{
		Code:    "INVALID_AGENT_DATA",
		Message: "invalid agent data",
	}
	ErrAgentSlugRequired = &Error{
		Code:    "AGENT_SLUG_REQUIRED",
		Message: "agent slug is required",
	}
	ErrAgentNameRequired = &Error{
		Code:    "AGENT_NAME_REQUIRED",
		Message: "agent name is required",
	}

	// Backward compatibility aliases
	ErrRoleNotFound     = ErrAgentNotFound
	ErrRoleAlreadyExists = ErrAgentAlreadyExists
	ErrRoleInUse        = ErrAgentInUse
	ErrInvalidRoleData  = ErrInvalidAgentData
	ErrRoleSlugRequired = ErrAgentSlugRequired
	ErrRoleNameRequired = ErrAgentNameRequired

	// Task errors
	ErrTaskNotFound = &Error{
		Code:    "TASK_NOT_FOUND",
		Message: "task not found",
	}
	ErrTaskAlreadyExists = &Error{
		Code:    "TASK_ALREADY_EXISTS",
		Message: "task already exists",
	}
	ErrInvalidTaskData = &Error{
		Code:    "INVALID_TASK_DATA",
		Message: "invalid task data",
	}
	ErrTaskTitleRequired = &Error{
		Code:    "TASK_TITLE_REQUIRED",
		Message: "task title is required",
	}
	ErrUnresolvedDependencies = &Error{
		Code:    "UNRESOLVED_DEPENDENCIES",
		Message: "task has unresolved dependencies",
	}
	ErrCircularDependency = &Error{
		Code:    "CIRCULAR_DEPENDENCY",
		Message: "circular dependency detected",
	}
	ErrTaskBlocked = &Error{
		Code:    "TASK_BLOCKED",
		Message: "task is blocked",
	}
	ErrTaskNotBlocked = &Error{
		Code:    "TASK_NOT_BLOCKED",
		Message: "task is not blocked",
	}
	ErrInvalidColumn = &Error{
		Code:    "INVALID_COLUMN",
		Message: "invalid column",
	}
	ErrCompletionSummaryRequired = &Error{
		Code:    "COMPLETION_SUMMARY_REQUIRED",
		Message: "completion summary is required (minimum 100 characters)",
	}
	ErrBlockedReasonRequired = &Error{
		Code:    "BLOCKED_REASON_REQUIRED",
		Message: "blocked reason is required (minimum 50 characters)",
	}
	ErrWontDoReasonRequired = &Error{
		Code:    "WONT_DO_REASON_REQUIRED",
		Message: "won't do reason is required (minimum 50 characters)",
	}
	ErrWontDoNotRequested = &Error{
		Code:    "WONT_DO_NOT_REQUESTED",
		Message: "won't do was not requested for this task",
	}
	ErrTaskNotInTodo = &Error{
		Code:    "TASK_NOT_IN_TODO",
		Message: "task is not in todo column",
	}
	ErrSummaryRequired = &Error{
		Code:    "SUMMARY_REQUIRED",
		Message: "summary is required",
	}
	ErrTaskNotInBlocked = &Error{
		Code:    "TASK_NOT_IN_BLOCKED",
		Message: "task is not in blocked column",
	}

	// Comment errors
	ErrCommentNotFound = &Error{
		Code:    "COMMENT_NOT_FOUND",
		Message: "comment not found",
	}
	ErrCommentNotEditable = &Error{
		Code:    "COMMENT_NOT_EDITABLE",
		Message: "comment cannot be edited",
	}
	ErrInvalidCommentData = &Error{
		Code:    "INVALID_COMMENT_DATA",
		Message: "invalid comment data",
	}
	ErrCommentContentRequired = &Error{
		Code:    "COMMENT_CONTENT_REQUIRED",
		Message: "comment content is required",
	}

	// Column errors
	ErrColumnNotFound = &Error{
		Code:    "COLUMN_NOT_FOUND",
		Message: "column not found",
	}

	// Dependency errors
	ErrDependencyNotFound = &Error{
		Code:    "DEPENDENCY_NOT_FOUND",
		Message: "dependency not found",
	}
	ErrDependencyAlreadyExists = &Error{
		Code:    "DEPENDENCY_ALREADY_EXISTS",
		Message: "dependency already exists",
	}
	ErrTaskHasDependents = &Error{
		Code:    "TASK_HAS_DEPENDENTS",
		Message: "task has dependent tasks that are not completed",
	}
	ErrCannotDependOnSelf = &Error{
		Code:    "CANNOT_DEPEND_ON_SELF",
		Message: "task cannot depend on itself",
	}
	ErrNoTasksAvailable = &Error{
		Code:    "NO_TASKS_AVAILABLE",
		Message: "no tasks available matching criteria",
	}

	// Skill errors
	ErrSkillNotFound = &Error{
		Code:    "SKILL_NOT_FOUND",
		Message: "skill not found",
	}
	ErrSkillAlreadyExists = &Error{
		Code:    "SKILL_ALREADY_EXISTS",
		Message: "skill already exists with this slug",
	}
	ErrSkillSlugRequired = &Error{
		Code:    "SKILL_SLUG_REQUIRED",
		Message: "skill slug is required",
	}
	ErrSkillNameRequired = &Error{
		Code:    "SKILL_NAME_REQUIRED",
		Message: "skill name is required",
	}
	ErrSkillInUse = &Error{
		Code:    "SKILL_IN_USE",
		Message: "skill is still assigned to one or more agents",
	}

	// Agent-project errors
	ErrAgentAlreadyInProject = &Error{
		Code:    "AGENT_ALREADY_IN_PROJECT",
		Message: "agent is already assigned to this project",
	}
	ErrAgentNotInProject = &Error{
		Code:    "AGENT_NOT_IN_PROJECT",
		Message: "agent is not assigned to this project",
	}
	ErrAgentHasTasks = &Error{
		Code:    "AGENT_HAS_TASKS",
		Message: "agent has tasks in this project; reassign or clear before removing",
	}

	ErrProjectsNotRelated = &Error{
		Code:    "PROJECTS_NOT_RELATED",
		Message: "source and target projects must share the same parent (be siblings) or have a direct parent-child relationship",
	}
	ErrFeatureNotFound = &Error{
		Code:    "FEATURE_NOT_FOUND",
		Message: "feature not found",
	}
	ErrFeatureNameRequired = &Error{
		Code:    "FEATURE_NAME_REQUIRED",
		Message: "feature name is required",
	}
	ErrInvalidFeatureStatus = &Error{
		Code:    "INVALID_FEATURE_STATUS",
		Message: "invalid feature status",
	}
	ErrFeatureNotInProject = &Error{
		Code:    "FEATURE_NOT_IN_PROJECT",
		Message: "feature does not belong to this project",
	}

	// Dockerfile errors
	ErrDockerfileNotFound = &Error{
		Code:    "DOCKERFILE_NOT_FOUND",
		Message: "dockerfile not found",
	}
	ErrDockerfileAlreadyExists = &Error{
		Code:    "DOCKERFILE_ALREADY_EXISTS",
		Message: "dockerfile already exists with this slug",
	}
	ErrDockerfileSlugRequired = &Error{
		Code:    "DOCKERFILE_SLUG_REQUIRED",
		Message: "dockerfile slug is required",
	}
	ErrDockerfileNameRequired = &Error{
		Code:    "DOCKERFILE_NAME_REQUIRED",
		Message: "dockerfile name is required",
	}
	ErrDockerfileVersionRequired = &Error{
		Code:    "DOCKERFILE_VERSION_REQUIRED",
		Message: "dockerfile version is required",
	}
	ErrDockerfileInUse = &Error{
		Code:    "DOCKERFILE_IN_USE",
		Message: "dockerfile is still assigned to one or more projects",
	}

	// Notification errors
	ErrNotificationNotFound = &Error{
		Code:    "NOTIFICATION_NOT_FOUND",
		Message: "notification not found",
	}
	ErrInvalidNotificationData = &Error{
		Code:    "INVALID_NOTIFICATION_DATA",
		Message: "invalid notification data",
	}
	ErrNotificationTitleRequired = &Error{
		Code:    "NOTIFICATION_TITLE_REQUIRED",
		Message: "notification title is required",
	}

	// Chat session errors
	ErrChatSessionNotFound = &Error{
		Code:    "CHAT_SESSION_NOT_FOUND",
		Message: "chat session not found",
	}

	// Aliases for backward compatibility
	ErrDependencyCycle  = ErrCircularDependency
	ErrNoAvailableTasks = ErrNoTasksAvailable
)
