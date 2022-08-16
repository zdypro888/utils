package utils

import (
	"bytes"
	"math"
	"math/rand"
	"strconv"
	"text/template"
)

// RandomTemplate 随机内容
type RandomTemplate struct {
	textContext []string
}

// Char 文本
func (rc *RandomTemplate) Char(count int) string {
	return GetRandomString(count)
}

// CharN 文本
func (rc *RandomTemplate) CharN(min int, max int) string {
	return GetRandomString(min + rand.Intn(max-min))
}

// Char 文本
func (rc *RandomTemplate) LChar(count int) string {
	return GetRandomLower(count)
}

// CharN 文本
func (rc *RandomTemplate) LCharN(min int, max int) string {
	return GetRandomLower(min + rand.Intn(max-min))
}

// Char 文本
func (rc *RandomTemplate) UChar(count int) string {
	return GetRandomLower(count)
}

// CharN 文本
func (rc *RandomTemplate) UCharN(min int, max int) string {
	return GetRandomLower(min + rand.Intn(max-min))
}

// Num 数字
func (rc *RandomTemplate) Num(count int) string {
	if count == 1 {
		return strconv.Itoa(rand.Intn(9))
	}
	min := int(math.Pow10(count - 1))
	return strconv.Itoa(min + rand.Intn(int(math.Pow10(count))-min-1))
}

// NumN 数字
func (rc *RandomTemplate) NumN(min int, max int) string {
	return strconv.Itoa(min + rand.Intn(max-min))
}

// In 随机选取
func (rc *RandomTemplate) In(texts ...string) string {
	return texts[rand.Intn(len(texts)-1)]
}

// Context 初始化文本 上下文
func (rc *RandomTemplate) Context(texts ...string) string {
	rc.textContext = texts
	return ""
}

// ContextOne 从上下文设置文本中随机选择一个
func (rc *RandomTemplate) ContextOne() string {
	return rc.textContext[rand.Intn(len(rc.textContext)-1)]
}

// RandomTemplateText 随机文本
func RandomTemplateText(text string) (string, error) {
	contentTpl, err := template.New("Random").Parse(text)
	if err != nil {
		return "", err
	}
	context := &RandomTemplate{}
	conio := &bytes.Buffer{}
	if err := contentTpl.Execute(conio, context); err != nil {
		return "", err
	}
	return conio.String(), nil
}
