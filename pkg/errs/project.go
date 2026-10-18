package errs

import "net/http"

// Error codes related to projects
var (
	ErrProjectIdInvalid                         = NewNormalError(NormalSubcategoryProject, 0, http.StatusBadRequest, "project id is invalid")
	ErrProjectNotFound                          = NewNormalError(NormalSubcategoryProject, 1, http.StatusBadRequest, "project not found")
	ErrProjectNameIsEmpty                       = NewNormalError(NormalSubcategoryProject, 2, http.StatusBadRequest, "project name is empty")
	ErrProjectNameAlreadyExists                 = NewNormalError(NormalSubcategoryProject, 3, http.StatusBadRequest, "project name already exists")
	ErrCannotUseHiddenProject                   = NewNormalError(NormalSubcategoryProject, 4, http.StatusBadRequest, "cannot use hidden project")
	ErrProjectCannotBeSetForBalanceModification = NewNormalError(NormalSubcategoryProject, 5, http.StatusBadRequest, "balance modification transaction cannot set project")
)
