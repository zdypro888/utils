package utils

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// JSONTime json时间
type JSONTime time.Time

// JSONInt 数字
type JSONInt int

// datatime format
const (
	jsonTimeFormart = `"2006-01-02 15:04:05"`
	RFC3339D        = "2006-01-02T15:04:05Z"
)

// UnmarshalJSON 到json时间
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	now, err := time.ParseInLocation(jsonTimeFormart, string(data), time.Local)
	if err != nil {
		return err
	}
	*t = JSONTime(now)
	return nil
}

// MarshalJSON 输出文本
func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format(jsonTimeFormart) + `"`), nil
}

// UnmarshalJSON 到json时间
func (t *JSONInt) UnmarshalJSON(data []byte) error {
	val, err := strconv.Atoi(strings.Trim(string(data), "\""))
	if err != nil {
		return err
	}
	*t = JSONInt(val)
	return nil
}

// MarshalJSON 输出文本
func (t JSONInt) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(t))), nil
}

// XMLCDATA xml标准CDATA类型
type XMLCDATA struct {
	Text string `xml:",cdata"`
}

// ConvertAnySlice 把 any 转换为 inteface{} 数组
func ConvertAnySlice[T any](slice []T) []any {
	r := make([]any, len(slice))
	for i, v := range slice {
		r[i] = v
	}
	return r
}

// ReadLinesFromReader 读取Reader中所有的行
func ReadLinesFromReader(r io.Reader, callback func(string) error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := callback(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// ReadLinesFromFile 读取文件中所有的行
func ReadLinesFromFile(path string, callback func(string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ReadLinesFromReader(f, callback)
}

// ReadLines 读取所有的行 如果i为文本型则是文件 如果为reader直接读取
func ReadLines(i any, callback func(string) error) error {
	switch v := i.(type) {
	case string:
		return ReadLinesFromFile(v, callback)
	case io.Reader:
		return ReadLinesFromReader(v, callback)
	}
	return errors.New("wrong i")
}

// ReadString 读取文本
func ReadString(r io.Reader) (string, error) {
	breader := bufio.NewReader(r)
	bytes, err := breader.ReadBytes(0)
	if err != nil {
		return "", err
	}
	return string(bytes[:len(bytes)-1]), nil
}

// WriteAll 将内容全部写出
func WriteAll(writer io.Writer, buf []byte) error {
	var err error
	var windex, wlen int
	blen := len(buf)
	for windex < blen {
		if wlen, err = writer.Write(buf[windex:]); err != nil {
			return err
		}
		if wlen == 0 {
			return io.ErrShortWrite
		}
		windex += wlen
	}
	return nil
}

// WriteString 将内容全部写出
func WriteString(writer io.Writer, str string) error {
	return WriteAll(writer, []byte(str))
}

// GetDateOnly 取得日期
func GetDateOnly(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// InterfaceIsNil 对象是否为nil
func InterfaceIsNil(a any) bool {
	defer func() { recover() }()
	return a == nil || reflect.ValueOf(a).IsNil()
}

// FileSHA1 取得file sha1
func FileSHA1(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	h := sha1.New()
	if _, err = io.Copy(h, file); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func getRandomWithMap(length int, randomMap string) string {
	bytes := []byte(randomMap)
	bytesLen := len(bytes)
	result := make([]byte, length)
	for i := range length {
		result[i] = bytes[rand.Intn(bytesLen)]
	}
	return string(result)
}

// GetRandomString 生成随机字符串
func GetRandomString(length int) string {
	return getRandomWithMap(length, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// GetRandomLower 生成随机字符串
func GetRandomLower(length int) string {
	return getRandomWithMap(length, "abcdefghijklmnopqrstuvwxyz")
}

// GetRandomLowerWithNumber 生成随机字符串
func GetRandomLowerWithNumber(length int) string {
	return getRandomWithMap(length, "0123456789abcdefghijklmnopqrstuvwxyz")
}

// GetRandomUpper 生成随机字符串
func GetRandomUpper(length int) string {
	return getRandomWithMap(length, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// GetRandomUpperWithNumber 生成随机字符串
func GetRandomUpperWithNumber(length int) string {
	return getRandomWithMap(length, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// PointerBuffer 指针到[]byte
func PointerBuffer(address uintptr, size int) []byte {
	var data []byte
	h := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
	h.Data = address
	h.Cap = size
	h.Len = h.Cap
	return data
}

// PathExist 文件是否存在
func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// SplitWithoutEmpty 分割但是去除空
func SplitWithoutEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, sep)
}

const (
	secondsPerMinute       = 60
	secondsPerHour         = 60 * secondsPerMinute
	secondsPerDay          = 24 * secondsPerHour
	unixToCocoa      int64 = (31*365 + 31/4 + 1) * secondsPerDay
)

// AppleCocoaTimestamp 苹果时间戳
func AppleCocoaTimestamp(t time.Time) int64 {
	return t.Unix() - unixToCocoa
}

// AppleCocoaTimestampNano 苹果时间戳 需要 /1000
func AppleCocoaTimestampNano(t time.Time) int64 {
	return t.UnixNano() - unixToCocoa*1e9
}

// AppleCocoaTimestamp 苹果时间戳
func AppleCocoaFromTimestamp(timespan int64) time.Time {
	return time.Unix(timespan+unixToCocoa, 0)
}

func HexToCArray(hexstr string, nlinec int) string {
	data, err := hex.DecodeString(strings.ReplaceAll(hexstr, " ", ""))
	if err != nil || len(data) == 0 {
		return ""
	}
	return DataCArray(data, nlinec)
}

func DataCArray(data []byte, nlinec int) string {
	writer := &bytes.Buffer{}
	for i, c := range data {
		writer.WriteString(fmt.Sprintf("%#02x, ", c))
		if nlinec > 0 && (i+1)%nlinec == 0 {
			writer.WriteString("\n")
		}
	}
	carray := writer.String()
	return strings.TrimSuffix(carray, ", ")
}

func BufferToInterger(data []byte, bit int) string {
	writer := &bytes.Buffer{}
	index := 0
	for len(data)-index > 0 {
		if len(data)-index < bit/8 {
			writer.WriteString(fmt.Sprintf("%x ", data[index:]))
			break
		}
		switch bit {
		case 8:
			writer.WriteString(fmt.Sprintf("%#02x ", data[index]))
		case 16:
			writer.WriteString(fmt.Sprintf("%#04x ", binary.LittleEndian.Uint16(data[index:])))
		case 32:
			writer.WriteString(fmt.Sprintf("%#08x ", binary.LittleEndian.Uint32(data[index:])))
		case 64:
			writer.WriteString(fmt.Sprintf("%#016x ", binary.LittleEndian.Uint64(data[index:])))
		}
		index += (bit / 8)
	}
	return writer.String()
}
