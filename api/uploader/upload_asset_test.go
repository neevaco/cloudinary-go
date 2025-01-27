package uploader_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudinary/cloudinary-go/api"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	"github.com/cloudinary/cloudinary-go/internal/cldtest"
)

var ctx = context.Background()
var uploadAPI, _ = uploader.New()

const largeImagePublicID = "go_test_large_image"
const largeImageSize = 5880138
const largeChunkSize = 5243000
const largeImageWidth = 1400
const largeImageHeight = 1400

func TestUploader_UploadLocalPath(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:              cldtest.PublicID,
		QualityAnalysis:       true,
		AccessibilityAnalysis: true,
		CinemagraphAnalysis:   true,
		Overwrite:             true,
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.ImageFilePath, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil || resp.PublicID != cldtest.PublicID {
		t.Error(resp)
	}
}

func TestUploader_UploadIOReader(t *testing.T) {
	file, err := os.Open(cldtest.ImageFilePath)
	if err != nil {
		t.Error(fmt.Printf("unable to open a file: %v\n", err))
	}

	defer api.DeferredClose(file)

	params := uploader.UploadParams{
		PublicID:              cldtest.PublicID,
		QualityAnalysis:       true,
		AccessibilityAnalysis: true,
		CinemagraphAnalysis:   true,
	}

	resp, err := uploadAPI.Upload(ctx, file, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil || resp.PublicID != cldtest.PublicID {
		t.Error(resp)
	}
}

func TestUploader_UploadURL(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:  cldtest.PublicID,
		Overwrite: true,
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.LogoURL, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil || resp.PublicID != cldtest.PublicID {
		t.Error(resp)
	}
}

func TestUploader_UploadVideoURL(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:     cldtest.PublicID,
		ResourceType: "video",
		Overwrite:    true,
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.VideoURL, params)

	if err != nil {
		t.Error(err)
	}
	if resp == nil || resp.PublicID != cldtest.PublicID || resp.Error.Message != "" {
		t.Error(resp)
	}
}

func TestUploader_UploadBase64Image(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:  cldtest.PublicID,
		Overwrite: true,
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.Base64Image, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil || resp.PublicID != cldtest.PublicID {
		t.Error(resp)
	}
}

func TestUploader_UploadAuthenticated(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:  cldtest.PublicID,
		Overwrite: true,
		Type:      api.Authenticated,
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.Base64Image, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil || resp.PublicID != cldtest.PublicID {
		t.Error(resp)
	}
}

func TestUploader_UploadLargeFile(t *testing.T) {
	uploadAPI.Config.API.ChunkSize = largeChunkSize

	largeImage := populateLargeImage()

	defer func() {
		err := os.Remove(largeImage)
		if err != nil {
			t.Error(err)
		}
	}()

	params := uploader.UploadParams{
		PublicID:  largeImagePublicID,
		Overwrite: true,
	}

	resp, err := uploadAPI.Upload(ctx, largeImage, params)

	if err != nil {
		t.Error(err)
	}

	// FIXME: destroy in teardown when available
	_, _ = uploadAPI.Destroy(ctx, uploader.DestroyParams{PublicID: largeImagePublicID})

	if resp == nil ||
		resp.PublicID != largeImagePublicID ||
		resp.Width != largeImageWidth ||
		resp.Height != largeImageHeight {
		t.Error(resp)
	}

}

func TestUploader_Timeout(t *testing.T) {
	var originalTimeout = uploadAPI.Config.API.Timeout

	uploadAPI.Config.API.Timeout = 0 // should timeout immediately

	_, err := uploadAPI.Upload(ctx, cldtest.LogoURL, uploader.UploadParams{})

	if err == nil || !strings.HasSuffix(err.Error(), "context deadline exceeded") {
		t.Error("Expected context timeout did not happen")
	}

	uploadAPI.Config.API.Timeout = originalTimeout
}

func TestUploader_UploadWithContext(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:  cldtest.PublicID,
		Overwrite: true,
		Context:   cldtest.CldContext,
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.LogoURL, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil {
		t.Error(resp)
	}

	assert.Equal(t, fmt.Sprintf("%v", cldtest.CldContext), fmt.Sprintf("%v", resp.Context["custom"]))
}

func TestUploader_UploadWithResponsiveBreakpoints(t *testing.T) {
	params := uploader.UploadParams{
		PublicID:              cldtest.PublicID,
		Overwrite:             true,
		ResponsiveBreakpoints: uploader.ResponsiveBreakpointsParams{{CreateDerived: false, Transformation: "a_90"}},
	}

	resp, err := uploadAPI.Upload(ctx, cldtest.LogoURL, params)

	if err != nil {
		t.Error(err)
	}

	if resp == nil {
		t.Error(resp)
	}

	assert.Len(t, resp.ResponsiveBreakpoints, 1)
	assert.Equal(t, "a_90", resp.ResponsiveBreakpoints[0].Transformation)

	eParams := uploader.ExplicitParams{
		PublicID: resp.PublicID,
		Type:     api.Upload,
		ResponsiveBreakpoints: uploader.ResponsiveBreakpointsParams{
			{CreateDerived: false, Transformation: "a_90"},
			{CreateDerived: false, Transformation: "a_45"},
		}}

	eResp, err := uploadAPI.Explicit(ctx, eParams)

	if err != nil {
		t.Error(err)
	}

	if eResp == nil {
		t.Error(resp)
	}

	assert.Len(t, eResp.ResponsiveBreakpoints, 2)
	assert.Equal(t, "a_90", eResp.ResponsiveBreakpoints[0].Transformation)
	assert.Equal(t, "a_45", eResp.ResponsiveBreakpoints[1].Transformation)
}

func populateLargeImage() string {
	head := "BMJ\xB9Y\x00\x00\x00\x00\x00\x8A\x00\x00\x00|\x00\x00\x00x\x05\x00\x00x\x05\x00\x00\x01\x00\x18\x00" +
		"\x00\x00\x00\x00\xC0\xB8Y\x00a\x0F\x00\x00a\x0F\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF" +
		"\x00\x00\xFF\x00\x00\xFF\x00\x00\x00\x00\x00\x00\xFFBGRs\x00\x00\x00\x00\x00\x00\x00\x00T\xB8\x1E" +
		"\xFC\x00\x00\x00\x00\x00\x00\x00\x00fff\xFC\x00\x00\x00\x00\x00\x00\x00\x00\xC4\xF5(\xFF\x00\x00\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

	tmpFile, err := ioutil.TempFile(cldtest.TestDataDir(), largeImagePublicID+".*.bmp")
	if err != nil {
		log.Fatal(err)
	}

	content := head + strings.Repeat("\xFF", largeImageSize-len(head))

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		_ = tmpFile.Close()
		log.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	return tmpFile.Name()
}

func TestColorWeightJSON(t *testing.T) {
	tests := []struct {
		JSON string
		CW   uploader.ColorWeight
	}{
		{
			JSON: `["#E4E4A8",71]`,
			CW:   uploader.ColorWeight{Color: "#E4E4A8", Weight: 71.0},
		},
		{
			JSON: `["#2F2F2F",8.7]`,
			CW:   uploader.ColorWeight{Color: "#2F2F2F", Weight: 8.7},
		},
		{
			JSON: `["brown",7.4]`,
			CW:   uploader.ColorWeight{Color: "brown", Weight: 7.4},
		},
	}
	for _, test := range tests {
		// Test json.Unmarshal
		var cw uploader.ColorWeight
		if err := json.Unmarshal([]byte(test.JSON), &cw); err != nil {
			t.Error(err)
		}
		if got, want := cw, test.CW; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		// Test json.Marshal
		b, err := json.Marshal(test.CW)
		if err != nil {
			t.Error(err)
		}
		if got, want := string(b), test.JSON; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestUploadResultJSON(t *testing.T) {
	tests := []struct {
		JSON   string
		Result uploader.UploadResult
	}{
		{
			JSON: `
{
	"asset_id": "c3f435bff0410515f8fdadb2a5037881",
	"public_id": "testimage",
	"version": 1645288244,
	"version_id": "ecc1001803c67e03780bb2e43a71314e",
	"signature": "910e43e4e41490d1bac6cc5309dde599c7edb933",
	"width": 600,
	"height": 600,
	"format": "png",
	"resource_type": "image",
	"created_at": "2022-02-19T16:30:44Z",
	"pages": 1,
	"bytes": 31543,
	"type": "upload",
	"etag": "a1e0cf45cf40c6a5e919ac6785d92d5b",
	"url": "http://foo.com/image/upload/v1645288244/testimage.png",
	"secure_url": "https://foo.com/image/upload/v1645288244/testimage.png",
	"colors": [
		[
			"#E4E4A8",
			71
		],
		[
			"#2F2F2F",
			8.7
		],
		[
			"#7A6241",
			7.8
		],
		[
			"#DEC39C",
			7.4
		]
	],
	"predominant": {
		"cloudinary": [
			[
				"yellow",
				71
			],
			[
				"black",
				8.7
			],
			[
				"brown",
				7.8
			],
			[
				"orange",
				7.4
			]
		],
		"google": [
			[
				"yellow",
				71
			],
			[
				"black",
				8.7
			],
			[
				"brown",
				7.8
			],
			[
				"orange",
				7.4
			]
		]
	},
	"phash": "31845b631e659ee9",
	"original_filename": "file"
}`,
			Result: uploader.UploadResult{
				AssetID:      "c3f435bff0410515f8fdadb2a5037881",
				PublicID:     "testimage",
				Version:      1645288244,
				VersionID:    "ecc1001803c67e03780bb2e43a71314e",
				Signature:    "910e43e4e41490d1bac6cc5309dde599c7edb933",
				Width:        600,
				Height:       600,
				Format:       "png",
				ResourceType: "image",
				CreatedAt:    time.Date(2022, time.February, 19, 16, 30, 44, 0, time.UTC),
				Pages:        1,
				Bytes:        31543,
				Type:         "upload",
				Etag:         "a1e0cf45cf40c6a5e919ac6785d92d5b",
				URL:          "http://foo.com/image/upload/v1645288244/testimage.png",
				SecureURL:    "https://foo.com/image/upload/v1645288244/testimage.png",
				Colors: []uploader.ColorWeight{
					{Color: "#E4E4A8", Weight: 71}, {Color: "#2F2F2F", Weight: 8.7}, {Color: "#7A6241", Weight: 7.8}, {Color: "#DEC39C", Weight: 7.4},
				},
				Predominant: map[string][]uploader.ColorWeight{
					"cloudinary": {{Color: "yellow", Weight: 71}, {Color: "black", Weight: 8.7}, {Color: "brown", Weight: 7.8}, {Color: "orange", Weight: 7.4}},
					"google":     {{Color: "yellow", Weight: 71}, {Color: "black", Weight: 8.7}, {Color: "brown", Weight: 7.8}, {Color: "orange", Weight: 7.4}},
				},
				Phash:            "31845b631e659ee9",
				OriginalFilename: "file",
			},
		},
	}
	for _, test := range tests {
		// Test json.Unmarshal
		var result uploader.UploadResult
		if err := json.Unmarshal([]byte(test.JSON), &result); err != nil {
			t.Error(err)
		}
		if got, want := result, test.Result; !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	}
}
