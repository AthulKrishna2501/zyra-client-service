package cloudinary

import (
	"context"
	"fmt"
	"log"

	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

var cld *cloudinary.Cloudinary

func InitCloudinary(cfg config.Config) {
	var err error
	cld, err = cloudinary.NewFromParams(cfg.CLOUD_NAME, cfg.CLOUD_API_KEY, cfg.CLOUD_SECRET)

	fmt.Printf("Cloudinary config: name=%s, key=%s, secret=%s", cfg.CLOUD_NAME, cfg.CLOUD_API_KEY, cfg.CLOUD_SECRET)
	if err != nil {
		log.Fatalf("Failed to init Cloudinary: %v", err)
	}
}

func UploadImage(filePath string) (string, *uploader.UploadResult, error) {
	if cld == nil {
		return "", nil, fmt.Errorf("cloudinary not initialized")
	}

	ctx := context.Background()
	resp, err := cld.Upload.Upload(ctx, filePath, uploader.UploadParams{
		Folder: "event_posters",
	})
	if err != nil {
		return "", nil, err
	}
	return resp.SecureURL, resp, nil
}

func GetImageURL(publicID string) (string, error) {
	if cld == nil {
		log.Println("Cloudinary not initialized")
		return "", nil
	}

	asset, err := cld.Image(publicID)
	if err != nil {
		log.Printf("Error generating image asset: %v\n", err)
		return "", err
	}

	return asset.String()
}

func DeleteImage(publicID string) error {
	if cld == nil {
		return fmt.Errorf("cloudinary not initialized")
	}

	ctx := context.Background()
	_, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID})
	return err
}
