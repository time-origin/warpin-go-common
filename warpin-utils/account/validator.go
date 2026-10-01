package account

import (
	"regexp"
)

// CheckMobile 检查是否为手机号
func CheckMobile(mobile string) bool {
	reg := `^1[3-9]\d{9}$`
	rgx := regexp.MustCompile(reg)
	return rgx.MatchString(mobile)
}

// CheckEmail 检查是否为邮箱
func CheckEmail(email string) bool {
	reg := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	rgx := regexp.MustCompile(reg)
	return rgx.MatchString(email)
}

// CheckIdCard 检查是否为身份证号 (简单校验)
func CheckIdCard(idCard string) bool {
	reg := `(^\d{15}$)|(^\d{18}$)|(^\d{17}(\d|X|x)$)`
	rgx := regexp.MustCompile(reg)
	return rgx.MatchString(idCard)
}

// CheckEmpCode 检查是否为员工工号 (假设为6位数字)
func CheckEmpCode(empCode string) bool {
	reg := `^\d{6}$`
	rgx := regexp.MustCompile(reg)
	return rgx.MatchString(empCode)
}
