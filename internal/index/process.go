package index

import (
	"context"
	"sync"

	"autofilterbot/internal/functions"
	"autofilterbot/internal/model"
	"autofilterbot/pkg/fileid"
	"github.com/amarnathcjd/gogram/telegram"
	"go.uber.org/zap"
)

func (o *Operation) MessageProcessor(ctx context.Context, c chan []telegram.Message, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case msgs, ok := <-c:
			if !ok {
				return
			}
			o.log.Debug("index: msgs to save received", zap.String("pid", o.ID), zap.Int("length", len(msgs)))

			var filesToSave []*model.File
			for _, m := range msgs {
				msg, ok := m.(*telegram.MessageObj)
				if !ok {
					// Silently skip non-MessageObj messages
					continue
				}

				if msg.Media == nil {
					// Silently skip messages with no media (e.g. simple text)
					continue
				}

				media, ok := msg.Media.(*telegram.MessageMediaDocument)
				if !ok {
					// Silently skip non-document messages (e.g. photos, webpage previews, location)
					continue
				}

				doc, ok := media.Document.(*telegram.DocumentObj)
				if !ok {
					// Silently skip if document object is empty/invalid
					continue
				}

				var (
					fileType            = model.FileTypeDocument
					fileIDType          = fileid.Document
					fileName            string
					unsupportedDocument bool
				)

				for _, attr := range doc.Attributes {
					switch a := attr.(type) {
					case *telegram.DocumentAttributeAnimated, *telegram.DocumentAttributeHasStickers, *telegram.DocumentAttributeImageSize, *telegram.DocumentAttributeSticker, *telegram.DocumentAttributeAudio:
						unsupportedDocument = true
					case *telegram.DocumentAttributeVideo:
						fileType = model.FileTypeVideo
						fileIDType = fileid.Video
					case *telegram.DocumentAttributeFilename:
						fileName = a.FileName
					}
				}

				if unsupportedDocument || fileName == "" {
					// Silently skip unsupported documents (animated, audio, sticker)
					continue
				}

				// Filter out non-video and non-archive documents, or garbage files
				if functions.IsGarbageFile(fileName) || !functions.HasVideoOrArchiveExtension(fileName) {
					continue
				}

				f := fileid.FileID{
					Type:          fileIDType,
					DC:            int(doc.DcID),
					ID:            doc.ID,
					AccessHash:    doc.AccessHash,
					FileReference: doc.FileReference,
				}

				fileID, err := fileid.EncodeFileID(f)
				if err != nil {
					o.log.Warn("encode file id failed", zap.String("pid", o.ID), zap.Int32("msg_id", msg.ID))
					o.mu.Lock()
					o.Failed++
					o.mu.Unlock()
					continue
				}

				file := &model.File{
					UniqueId: functions.RandString(15),
					FileId:   fileID,
					FileName: functions.RemoveSymbols(functions.CleanPromoFromName(functions.RemoveExtension(fileName))),
					FileType: fileType,
					FileSize: int64(doc.Size),
					Time:     int64(msg.Date),
				}
				filesToSave = append(filesToSave, file)
			}

			if len(filesToSave) > 0 {
				err := o.db.BulkSaveFiles(filesToSave)
				o.mu.Lock()
				if err != nil {
					o.log.Warn("index: bulk save failed", zap.Error(err), zap.String("pid", o.ID))
					o.Failed += len(filesToSave)
				} else {
					o.Saved += len(filesToSave)
				}
				o.mu.Unlock()
			}
		case <-ctx.Done():
			return
		case <-o.completedSignal:
			return
		}
	}
}
