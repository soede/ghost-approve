package botErrors

import "errors"

var FileLockedError = errors.New("file in use")
var NotInUserState = errors.New("not found in userState")

var NotFoundUsers = errors.New("not found users for notification")
var NotFoundApprovalsForReports = errors.New("Не найдено заврешенных апрувов для генерации отчетов")

var ErrHoursBelowMinimum = errors.New("количество часов не может быть меньше 1")

var ErrHoursExceedLimit = errors.New("количество часов не может превышать 2 месяца")

var ErrDateInPast = errors.New("указанная дата уже прошла")

var ErrApprovalIsNotRelevant = errors.New("approval is no longer relevant")

var ErrAlreadyHasResponse = errors.New("user already in ApproveBy or RejectBy")

var ErrNoAccess = errors.New("No access")

var ErrNotFound = errors.New("approve not found")

var ErrTaskIsNil = errors.New("Task is nil")
