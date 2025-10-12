package template

import "fmt"

func GreetingTemplate(name string) string {
	template := fmt.Sprintf(`
		<html>
        <body>
            <h2>Email xin chào</h2>
            <p>Kính gửi %s,</p>
            <p>Cảm ơn bạn đã tin tưởng và lựa chọn Agrisa.</p>
            <br>
            <p>Trân trọng,<br>Đội ngũ Agrisa</p>
        </body>
        </html>
		`, name)
	return template
}
