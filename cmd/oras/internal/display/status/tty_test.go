/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package status

import (
	"errors"
	"os"
	"sync"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/internal/testutils"
)

func TestTTYPushHandler_OnFileLoading(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdout, mockFetcher.Fetcher)
	if ph.OnFileLoading("test") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
}

func TestTTYPushHandler_OnEmptyArtifact(t *testing.T) {
	ph := NewTTYAttachHandler(os.Stdout, mockFetcher.Fetcher)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
}

func TestTTYPushHandler_TrackTarget_invalidTTY(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdin, mockFetcher.Fetcher)
	if _, _, err := ph.TrackTarget(nil); err == nil {
		t.Error("TrackTarget() should return an error for non-tty file")
	}
}

func TestTTYPullHandler_OnNodeDownloading(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeDownloading(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeDownloading() should not return an error")
	}
}

func TestTTYPullHandler_OnNodeDownloaded(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeDownloaded(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeDownloaded() should not return an error")
	}
}

func TestTTYPullHandler_OnNodeProcessing(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeProcessing(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeProcessing() should not return an error")
	}
}

func TestTTYPushHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle])
	ph := &TTYPushHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := ph.PostCopy(ctx, fetcher.OciImage); err != nil {
		t.Errorf("unexpected error from PostCopy(): %v", err)
	}
}

func TestTTYPushHandler_PostCopy_errGetSuccessor(t *testing.T) {
	errorFetcher := testutils.NewErrorFetcher()
	ph := NewTTYPushHandler(nil, errorFetcher)
	err := ph.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})
	if err.Error() != errorFetcher.ExpectedError.Error() {
		t.Errorf("PostCopy() should return expected error got %v", err.Error())
	}
}

func TestTTYPushHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	ph := &TTYPushHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := ph.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestNewTTYBackupHandler(t *testing.T) {
	handler := NewTTYBackupHandler(os.Stdout, nil)
	if handler == nil {
		t.Error("NewTTYBackupHandler() should not return nil")
	}
}

func TestTTYBackupHandler_StartTracking_invalidTTY(t *testing.T) {
	bh := NewTTYBackupHandler(os.Stdin, nil)
	gt := memory.New()
	if _, err := bh.StartTracking(gt); err == nil {
		t.Error("StartTracking() should return an error for non-tty file")
	}
}

func TestTTYBackupHandler_OnCopySkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	bh := &TTYBackupHandler{
		tracked:   &testutils.PromptDiscarder{}, // Keep PromptDiscarder here for Report method
		committed: &sync.Map{},
		fetcher:   fetcher.Fetcher,
	}
	if err := bh.OnCopySkipped(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("OnCopySkipped() should not return an error: %v", err)
	}

	// Verify that the descriptor is stored in the committed map
	if _, ok := bh.committed.Load(fetcher.ImageLayer.Digest.String()); !ok {
		t.Error("OnCopySkipped() should store the descriptor in the committed map")
	}
}

func TestTTYBackupHandler_PreCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	bh := &TTYBackupHandler{}
	if err := bh.PreCopy(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("PreCopy() should not return an error: %v", err)
	}
}

func TestTTYBackupHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle])
	bh := &TTYBackupHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := bh.PostCopy(ctx, fetcher.OciImage); err != nil {
		t.Errorf("unexpected error from PostCopy(): %v", err)
	}
}

func TestTTYBackupHandler_PostCopy_errGetSuccessor(t *testing.T) {
	errorFetcher := testutils.NewErrorFetcher()
	prompt := &testutils.PromptDiscarder{}
	bh := &TTYBackupHandler{
		tracked:   prompt,
		committed: &sync.Map{},
		fetcher:   errorFetcher,
	}

	err := bh.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})

	if err == nil || err.Error() != errorFetcher.ExpectedError.Error() {
		t.Errorf("PostCopy() should return expected error got %v", err.Error())
	}
}

func TestTTYBackupHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	bh := &TTYBackupHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := bh.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestNewTTYRestoreHandler(t *testing.T) {
	handler := NewTTYRestoreHandler(os.Stdout, nil)
	if handler == nil {
		t.Error("NewTTYRestoreHandler() should not return nil")
	}
}

func TestTTYRestoreHandler_StartTracking_invalidTTY(t *testing.T) {
	rh := NewTTYRestoreHandler(os.Stdin, nil)
	gt := memory.New()
	if _, err := rh.StartTracking(gt); err == nil {
		t.Error("StartTracking() should return an error for non-tty file")
	}
}

func TestTTYRestoreHandler_OnCopySkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	rh := &TTYRestoreHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: &sync.Map{},
		fetcher:   fetcher.Fetcher,
	}
	if err := rh.OnCopySkipped(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("OnCopySkipped() should not return an error: %v", err)
	}

	// Verify that the descriptor is stored in the committed map
	if _, ok := rh.committed.Load(fetcher.ImageLayer.Digest.String()); !ok {
		t.Error("OnCopySkipped() should store the descriptor in the committed map")
	}
}

func TestTTYRestoreHandler_PreCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	rh := &TTYRestoreHandler{}
	if err := rh.PreCopy(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("PreCopy() should not return an error: %v", err)
	}
}

func TestTTYRestoreHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle])
	rh := &TTYRestoreHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := rh.PostCopy(ctx, fetcher.OciImage); err != nil {
		t.Errorf("unexpected error from PostCopy(): %v", err)
	}
}

func TestTTYRestoreHandler_PostCopy_errGetSuccessor(t *testing.T) {
	errorFetcher := testutils.NewErrorFetcher()
	prompt := &testutils.PromptDiscarder{}
	rh := &TTYRestoreHandler{
		tracked:   prompt,
		committed: &sync.Map{},
		fetcher:   errorFetcher,
	}

	err := rh.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})

	if err == nil || err.Error() != errorFetcher.ExpectedError.Error() {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestTTYRestoreHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	rh := &TTYRestoreHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := rh.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestTTYPushHandler_OnCopySkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	ph := &TTYPushHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: &sync.Map{},
		fetcher:   fetcher.Fetcher,
	}
	if err := ph.OnCopySkipped(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("OnCopySkipped() should not return an error: %v", err)
	}

	// Verify that the descriptor is stored in the committed map
	if _, ok := ph.committed.Load(fetcher.ImageLayer.Digest.String()); !ok {
		t.Error("OnCopySkipped() should store the descriptor in the committed map")
	}
}

func TestTTYPushHandler_OnCopySkipped_errReport(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	wantedError := errors.New("report error")
	ph := &TTYPushHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: &sync.Map{},
		fetcher:   fetcher.Fetcher,
	}
	if err := ph.OnCopySkipped(ctx, fetcher.ImageLayer); err != wantedError {
		t.Errorf("OnCopySkipped() should return expected error got %v", err)
	}
}

func TestTTYPushHandler_PreCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	ph := &TTYPushHandler{}
	if err := ph.PreCopy(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("PreCopy() should not return an error: %v", err)
	}
}

func TestTTYPullHandler_OnNodeRestored(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	ph := &TTYPullHandler{
		tracked: &testutils.PromptDiscarder{},
	}
	if err := ph.OnNodeRestored(fetcher.ImageLayer); err != nil {
		t.Errorf("OnNodeRestored() should not return an error: %v", err)
	}
}

func TestTTYPullHandler_OnNodeRestored_errReport(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	wantedError := errors.New("report error")
	ph := &TTYPullHandler{
		tracked: testutils.NewErrorPrompt(wantedError),
	}
	if err := ph.OnNodeRestored(fetcher.ImageLayer); err != wantedError {
		t.Errorf("OnNodeRestored() should return expected error got %v", err)
	}
}

func TestTTYPullHandler_OnNodeSkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	ph := &TTYPullHandler{
		tracked: &testutils.PromptDiscarder{},
	}
	if err := ph.OnNodeSkipped(fetcher.ImageLayer); err != nil {
		t.Errorf("OnNodeSkipped() should not return an error: %v", err)
	}
}

func TestTTYPullHandler_OnNodeSkipped_errReport(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	wantedError := errors.New("report error")
	ph := &TTYPullHandler{
		tracked: testutils.NewErrorPrompt(wantedError),
	}
	if err := ph.OnNodeSkipped(fetcher.ImageLayer); err != wantedError {
		t.Errorf("OnNodeSkipped() should return expected error got %v", err)
	}
}

func TestTTYCopyHandler_StartTracking_invalidTTY(t *testing.T) {
	ch := NewTTYCopyHandler(os.Stdin)
	gt := memory.New()
	if _, err := ch.StartTracking(gt); err == nil {
		t.Error("StartTracking() should return an error for non-tty file")
	}
}

func TestTTYCopyHandler_PostCopy_success(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	store := memory.New()
	ch := &TTYCopyHandler{
		tracked:   &testutils.PromptDiscarder{GraphTarget: store},
		committed: sync.Map{},
	}
	// Use Config descriptor which has no successors, so no fetch is needed for successors
	if err := ch.PostCopy(ctx, fetcher.Config); err != nil {
		t.Errorf("PostCopy() should not return an error: %v", err)
	}
}

func TestTTYCopyHandler_PostCopy_errGetSuccessor(t *testing.T) {
	// Use an empty memory store - fetching from it will return "not found" error
	store := memory.New()
	ch := &TTYCopyHandler{
		tracked:   &testutils.PromptDiscarder{GraphTarget: store},
		committed: sync.Map{},
	}

	// Using a bogus descriptor with manifest media type causes FilteredSuccessors
	// to try to fetch and parse the manifest, which will fail (not found)
	err := ch.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})

	if err == nil {
		t.Error("PostCopy() should return an error for invalid manifest")
	}
}

func TestTTYCopyHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := sync.Map{}
	// Store a different title to trigger skipped reporting for the layer
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	errorPrompt := testutils.NewErrorPrompt(wantedError)
	// Set the GraphTarget to the mock fetcher's store so it can fetch the manifest
	errorPrompt.GraphTarget = fetcher.Fetcher.(oras.GraphTarget)
	ch := &TTYCopyHandler{
		tracked:   errorPrompt,
		committed: committed,
	}
	if err := ch.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestNewTTYBlobPushHandler(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Size:      100,
	}
	handler := NewTTYBlobPushHandler(os.Stdout, desc)
	if handler == nil {
		t.Error("NewTTYBlobPushHandler() should not return nil")
	}
}

func TestTTYBlobPushHandler_StartTracking_invalidTTY(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Size:      100,
	}
	bph := NewTTYBlobPushHandler(os.Stdin, desc)
	gt := memory.New()
	if _, err := bph.StartTracking(gt); err == nil {
		t.Error("StartTracking() should return an error for non-tty file")
	}
}

func TestTTYBlobPushHandler_OnBlobExists(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	bph := &TTYBlobPushHandler{
		desc:    fetcher.ImageLayer,
		tracked: &testutils.PromptDiscarder{},
	}
	if err := bph.OnBlobExists(); err != nil {
		t.Errorf("OnBlobExists() should not return an error: %v", err)
	}
}

func TestTTYBlobPushHandler_OnBlobExists_errReport(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	wantedError := errors.New("report error")
	bph := &TTYBlobPushHandler{
		desc:    fetcher.ImageLayer,
		tracked: testutils.NewErrorPrompt(wantedError),
	}
	if err := bph.OnBlobExists(); err != wantedError {
		t.Errorf("OnBlobExists() should return expected error got %v", err)
	}
}

func TestTTYBlobPushHandler_OnBlobUploading(t *testing.T) {
	bph := &TTYBlobPushHandler{}
	if err := bph.OnBlobUploading(); err != nil {
		t.Errorf("OnBlobUploading() should not return an error: %v", err)
	}
}

func TestTTYBlobPushHandler_OnBlobUploaded(t *testing.T) {
	bph := &TTYBlobPushHandler{}
	if err := bph.OnBlobUploaded(); err != nil {
		t.Errorf("OnBlobUploaded() should not return an error: %v", err)
	}
}

// nopCloser is a simple mock closer that does nothing
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

func TestTTYBackupHandler_StopTracking(t *testing.T) {
	bh := &TTYBackupHandler{
		tracked: &testutils.PromptDiscarder{Closer: nopCloser{}},
	}
	if err := bh.StopTracking(); err != nil {
		t.Errorf("StopTracking() should not return an error: %v", err)
	}
}

func TestTTYRestoreHandler_StopTracking(t *testing.T) {
	rh := &TTYRestoreHandler{
		tracked: &testutils.PromptDiscarder{Closer: nopCloser{}},
	}
	if err := rh.StopTracking(); err != nil {
		t.Errorf("StopTracking() should not return an error: %v", err)
	}
}

func TestTTYBlobPushHandler_StopTracking(t *testing.T) {
	bph := &TTYBlobPushHandler{
		tracked: &testutils.PromptDiscarder{Closer: nopCloser{}},
	}
	if err := bph.StopTracking(); err != nil {
		t.Errorf("StopTracking() should not return an error: %v", err)
	}
}
