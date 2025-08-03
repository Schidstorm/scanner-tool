package server

// type zipInputFile struct {
// 	*zip.File
// 	metadata map[string]string
// }

// func (z *zipInputFile) Metadata() map[string]string {
// 	return z.metadata
// }

// func forAllFilesInZip(zipFile filequeue.QueueFile, handler func(f InputFile) error) error {
// 	fileSize, err := zipFile.Size()
// 	if err != nil {
// 		return err
// 	}

// 	zipReader, err := zip.NewReader(zipFile, fileSize)
// 	if err != nil {
// 		return err
// 	}

// 	metadataMap := make(map[string]*queueoutputcreator.Metadata)
// 	for _, file := range zipReader.File {
// 		if file.FileInfo().IsDir() {
// 			continue
// 		}

// 		if strings.HasPrefix(file.Name, ".metadata.") {
// 			zfContent, err := readZipFile(file)
// 			if err != nil {
// 				return err
// 			}
// 			metadataMap[file.Name] = queueoutputcreator.DeserializeMetadata(zfContent)
// 		}
// 	}

// 	for _, file := range zipReader.File {
// 		if file.FileInfo().IsDir() {
// 			continue
// 		}
// 		if strings.HasPrefix(file.Name, ".metadata.") {
// 			continue
// 		}

// 		var metadata map[string]string
// 		if m, ok := metadataMap[".metadata."+file.Name]; ok {
// 			metadata = m.ToMap()
// 		} else {
// 			metadata = make(map[string]string)
// 		}

// 		err = handler(&zipInputFile{
// 			File:     file,
// 			metadata: metadata,
// 		})
// 		if err != nil {
// 			return err
// 		}

// 	}

// 	return nil
// }
