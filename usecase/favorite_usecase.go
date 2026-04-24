package usecase

import (
	"context"
	"tripsync-backend/dto"
	"tripsync-backend/repository"
)

type FavoriteUseCase interface {
	ToggleFavorite(ctx context.Context, userID uint, groupID uint) (bool, error)
	GetFavorites(ctx context.Context, userID uint) ([]dto.GroupWithRole, error)
}

type favoriteUseCaseImpl struct {
	repo repository.FavoriteRepository
}

func NewFavoriteUseCase(repo repository.FavoriteRepository) FavoriteUseCase {
	return &favoriteUseCaseImpl{repo: repo}
}

func (u *favoriteUseCaseImpl) ToggleFavorite(ctx context.Context, userID uint, groupID uint) (bool, error) {
	return u.repo.Toggle(ctx, userID, groupID)
}

func (u *favoriteUseCaseImpl) GetFavorites(ctx context.Context, userID uint) ([]dto.GroupWithRole, error) {
	favorites, err := u.repo.GetFavoritesByUser(ctx, userID)
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
