package usecase

import (
	"tripsync-backend/dto"
	"tripsync-backend/repository"
)

type FavoriteUseCase interface {
	ToggleFavorite(userID uint, groupID uint) (bool, error)
	GetFavorites(userID uint) ([]dto.GroupWithRole, error)
}

type favoriteUseCaseImpl struct {
	repo repository.FavoriteRepository
}

func NewFavoriteUseCase(repo repository.FavoriteRepository) FavoriteUseCase {
	return &favoriteUseCaseImpl{repo: repo}
}

func (u *favoriteUseCaseImpl) ToggleFavorite(userID uint, groupID uint) (bool, error) {
	return u.repo.Toggle(userID, groupID)
}

func (u *favoriteUseCaseImpl) GetFavorites(userID uint) ([]dto.GroupWithRole, error) {
	favorites, err := u.repo.GetFavoritesByUser(userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.GroupWithRole, 0, len(favorites))
	for _, fav := range favorites {
		result = append(result, dto.GroupWithRole{
			Group: fav.Group,
			Role:  "FAVORITE",
		})
	}
	return result, nil
}
