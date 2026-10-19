package errs

import "net/http"

// Error codes related to budget alerts
var (
	ErrBudgetAlertIdInvalid = NewNormalError(NormalSubcategoryBudgetAlert, 0, http.StatusBadRequest, "budget alert id is invalid")
)
