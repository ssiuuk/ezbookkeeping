package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// ProjectService represents project service
type ProjectService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a project service singleton instance
var (
	Projects = &ProjectService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetAllProjectsByUid returns all project models of user
func (s *ProjectService) GetAllProjectsByUid(c core.Context, uid int64) ([]*models.Project, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var projects []*models.Project
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).Find(&projects)

	return projects, err
}

// GetProjectByProjectId returns a project model according to project id
func (s *ProjectService) GetProjectByProjectId(c core.Context, uid int64, projectId int64) (*models.Project, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if projectId <= 0 {
		return nil, errs.ErrProjectIdInvalid
	}

	project := &models.Project{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(projectId).Where("uid=? AND deleted=?", uid, false).Get(project)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrProjectNotFound
	}

	return project, nil
}

// GetProjectTransactionStatistics returns the transaction count and time range of every project of user
func (s *ProjectService) GetProjectTransactionStatistics(c core.Context, uid int64, projectId int64) (map[int64]*models.ProjectTransactionStatistic, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	condition := "uid=? AND deleted=? AND type<>? AND project_id>?"
	conditionParams := []any{uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_IN, 0}

	if projectId > 0 {
		condition = "uid=? AND deleted=? AND type<>? AND project_id=?"
		conditionParams = []any{uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_IN, projectId}
	}

	var statistics []*models.ProjectTransactionStatistic
	err := s.UserDataDB(uid).NewSession(c).Table(&models.Transaction{}).
		Select("project_id, COUNT(*) AS transaction_count, MIN(transaction_time) AS min_transaction_time, MAX(transaction_time) AS max_transaction_time").
		Where(condition, conditionParams...).
		GroupBy("project_id").
		Find(&statistics)

	if err != nil {
		return nil, err
	}

	result := make(map[int64]*models.ProjectTransactionStatistic, len(statistics))

	for i := 0; i < len(statistics); i++ {
		result[statistics[i].ProjectId] = statistics[i]
	}

	return result, nil
}

// GetMaxDisplayOrder returns the max display order
func (s *ProjectService) GetMaxDisplayOrder(c core.Context, uid int64) (int32, error) {
	if uid <= 0 {
		return 0, errs.ErrUserIdInvalid
	}

	project := &models.Project{}
	has, err := s.UserDataDB(uid).NewSession(c).Cols("uid", "deleted", "display_order").Where("uid=? AND deleted=?", uid, false).OrderBy("display_order desc").Limit(1).Get(project)

	if err != nil {
		return 0, err
	}

	if has {
		return project.DisplayOrder, nil
	}

	return 0, nil
}

// CreateProject saves a new project model to database
func (s *ProjectService) CreateProject(c core.Context, project *models.Project) error {
	if project.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	exists, err := s.ExistsProjectName(c, project.Uid, project.Name)

	if err != nil {
		return err
	} else if exists {
		return errs.ErrProjectNameAlreadyExists
	}

	project.ProjectId = s.GenerateUuid(uuid.UUID_TYPE_PROJECT)

	if project.ProjectId < 1 {
		return errs.ErrSystemIsBusy
	}

	now := time.Now().Unix()
	project.Deleted = false
	project.CreatedUnixTime = now
	project.UpdatedUnixTime = now

	return s.UserDataDB(project.Uid).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(project)
		return err
	})
}

// ModifyProject saves an existed project model to database
func (s *ProjectService) ModifyProject(c core.Context, project *models.Project, nameChanged bool) error {
	if project.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if nameChanged {
		exists, err := s.ExistsProjectName(c, project.Uid, project.Name)

		if err != nil {
			return err
		} else if exists {
			return errs.ErrProjectNameAlreadyExists
		}
	}

	project.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(project.Uid).DoTransaction(c, func(sess *xorm.Session) error {
		updatedRows, err := sess.ID(project.ProjectId).Cols("name", "comment", "updated_unix_time").Where("uid=? AND deleted=?", project.Uid, false).Update(project)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrProjectNotFound
		}

		return nil
	})
}

// HideProject updates hidden field of given projects
func (s *ProjectService) HideProject(c core.Context, uid int64, ids []int64, hidden bool) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	updateModel := &models.Project{
		Hidden:          hidden,
		UpdatedUnixTime: time.Now().Unix(),
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		updatedRows, err := sess.Cols("hidden", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).In("project_id", ids).Update(updateModel)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrProjectNotFound
		}

		return nil
	})
}

// ModifyProjectDisplayOrders updates display order of given projects
func (s *ProjectService) ModifyProjectDisplayOrders(c core.Context, uid int64, projects []*models.Project) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()

	for i := 0; i < len(projects); i++ {
		projects[i].UpdatedUnixTime = now
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		for i := 0; i < len(projects); i++ {
			project := projects[i]
			updatedRows, err := sess.ID(project.ProjectId).Cols("display_order", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(project)

			if err != nil {
				return err
			} else if updatedRows < 1 {
				return errs.ErrProjectNotFound
			}
		}

		return nil
	})
}

// DeleteProject deletes an existed project from database, transactions of this project are kept but no longer linked to it
func (s *ProjectService) DeleteProject(c core.Context, uid int64, projectId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()

	updateModel := &models.Project{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.ID(projectId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrProjectNotFound
		}

		transactionUpdateModel := &models.Transaction{
			ProjectId:       0,
			UpdatedUnixTime: now,
		}

		_, err = sess.Cols("project_id", "updated_unix_time").Where("uid=? AND project_id=?", uid, projectId).Update(transactionUpdateModel)

		return err
	})
}

// DeleteAllProjects deletes all existed projects from database
func (s *ProjectService) DeleteAllProjects(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	updateModel := &models.Project{
		Deleted:         true,
		DeletedUnixTime: time.Now().Unix(),
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)
		return err
	})
}

// ExistsProjectName returns whether the given project name exists
func (s *ProjectService) ExistsProjectName(c core.Context, uid int64, name string) (bool, error) {
	if name == "" {
		return false, errs.ErrProjectNameIsEmpty
	}

	return s.UserDataDB(uid).NewSession(c).Cols("name").Where("uid=? AND deleted=? AND name=?", uid, false, name).Exist(&models.Project{})
}

// GetProjectMapByList returns a project map by a list
func (s *ProjectService) GetProjectMapByList(projects []*models.Project) map[int64]*models.Project {
	projectMap := make(map[int64]*models.Project, len(projects))

	for i := 0; i < len(projects); i++ {
		projectMap[projects[i].ProjectId] = projects[i]
	}

	return projectMap
}
