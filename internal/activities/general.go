package activities

import (
	"context"
	"fmt"

	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/dto"
	"github.com/zibbp/ganymede/internal/utils"
)

func CreateDirectory(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, err := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodCreateFolder(utils.Running).Save(ctx)
	if err != nil {
		return err
	}

	err = utils.CreateFolder(fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName))
	if err != nil {

		_, err := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodCreateFolder(utils.Failed).Save(ctx)
		if err != nil {
			return err
		}
		return err
	}

	_, err = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodCreateFolder(utils.Success).Save(ctx)
	if err != nil {
		return err
	}

	return nil
}
