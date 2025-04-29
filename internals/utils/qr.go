package utils

import (
	"encoding/base64"

	"github.com/skip2/go-qrcode"
)

func GenerateQRCode(data string) string {
	qr, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		return ""
	}
	base64Image := base64.StdEncoding.EncodeToString(qr)
	return "data:image/png;base64," + base64Image
}
