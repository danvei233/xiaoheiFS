package avatar

import "fmt"

const (
	qqAvatarURL = "https://q1.qlogo.cn/g?b=qq&nk=%s&s=%d"
)

func GetQQAvatarURL(qq string, size int) string {
	if size <= 0 {
		size = 100
	}
	return fmt.Sprintf(qqAvatarURL, qq, size)
}

func GetQQAvatarURLDefault(qq string) string {
	return GetQQAvatarURL(qq, 100)
}
