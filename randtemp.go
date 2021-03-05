package utils

import (
	"bytes"
	"math"
	"math/rand"
	"strconv"
	"text/template"
)

//RandomTemplate 随机内容
type RandomTemplate struct {
	textContext []string
}

//RandChar 文本
func (rc *RandomTemplate) RandChar(count int) string {
	return GetRandomLower(count)
}

//RandCharN 文本
func (rc *RandomTemplate) RandCharN(min int, max int) string {
	return GetRandomLower(min + rand.Intn(max-min))
}

//RandNum 数字
func (rc *RandomTemplate) RandNum(count int) string {
	if count == 1 {
		return strconv.Itoa(rand.Intn(9))
	}
	min := int(math.Pow10(count - 1))
	return strconv.Itoa(min + rand.Intn(int(math.Pow10(count))-min-1))
}

//RandNumN 数字
func (rc *RandomTemplate) RandNumN(min int, max int) string {
	return strconv.Itoa(min + rand.Intn(max-min))
}

//RandIn 随机选取
func (rc *RandomTemplate) RandIn(texts ...string) string {
	return texts[rand.Intn(len(texts)-1)]
}

//TextContext 初始化文本 上下文
func (rc *RandomTemplate) TextContext(texts ...string) string {
	rc.textContext = texts
	return ""
}

//TextIn 从上下文设置文本中随机选择一个
func (rc *RandomTemplate) TextIn() string {
	return rc.textContext[rand.Intn(len(rc.textContext)-1)]
}

//RandomTemplateText 随机文本
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
	return string(conio.Bytes()), nil
}
