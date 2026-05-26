package main

// func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
// 	// Generates presigned URL to access an object in a private s3 bucket
// 	presignClient := s3.NewPresignClient(s3Client)
// 	params := s3.GetObjectInput{
// 		Bucket: &bucket,
// 		Key:    &key,
// 	}
// 	presignedRequest, err := presignClient.PresignGetObject(context.Background(), &params, s3.WithPresignExpires(expireTime))
// 	if err != nil {
// 		return "", err
// 	}
// 	return presignedRequest.URL, nil
// }

// func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
// 	bucketKey := strings.Split(*video.VideoURL, ",")
// 	fmt.Println("In dbVideoToSignedVideo, video URL: ", *video.VideoURL)
// 	fmt.Println("In dbVideoToSignedVideo, bucketKey: ", bucketKey)

// 	presignedURL, err := generatePresignedURL(cfg.s3Client, bucketKey[0], bucketKey[1], time.Hour)
// 	fmt.Println("In dbVideoToSignedVideo, presigned URL: ", presignedURL)
// 	if err != nil {
// 		return database.Video{}, err
// 	}
// 	video.VideoURL = &presignedURL
// 	fmt.Println("In dbVideoToSignedVideo, presigned *video.VideoURL: ", *video.VideoURL)
// 	return video, nil
// }
