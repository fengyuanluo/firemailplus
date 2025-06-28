package encoding

import (
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

// EncodingDetector 编码检测器接口
type EncodingDetector interface {
	DetectEncoding(data []byte) (string, float64, error)
	DetectEncodingFromString(text string) (string, float64, error)
	GetSupportedEncodings() []string
}

// EncodingConverter 编码转换器接口
type EncodingConverter interface {
	ConvertToUTF8(data []byte, sourceEncoding string) ([]byte, error)
	ConvertFromUTF8(data []byte, targetEncoding string) ([]byte, error)
	ConvertString(text, sourceEncoding, targetEncoding string) (string, error)
	IsValidUTF8(data []byte) bool
}

// EncodingService 编码服务接口
type EncodingService interface {
	EncodingDetector
	EncodingConverter
	AutoConvertToUTF8(data []byte) ([]byte, string, error)
	AutoConvertStringToUTF8(text string) (string, string, error)
}

// StandardEncodingDetector 标准编码检测器
type StandardEncodingDetector struct {
	// 支持的编码映射
	encodings map[string]encoding.Encoding
	
	// 编码检测规则
	detectionRules []DetectionRule
}

// DetectionRule 编码检测规则
type DetectionRule struct {
	Name       string
	Encoding   string
	Confidence float64
	Detector   func([]byte) (bool, float64)
}

// NewStandardEncodingDetector 创建标准编码检测器
func NewStandardEncodingDetector() EncodingDetector {
	detector := &StandardEncodingDetector{
		encodings: make(map[string]encoding.Encoding),
	}
	
	detector.initializeEncodings()
	detector.initializeDetectionRules()
	
	return detector
}

// initializeEncodings 初始化编码映射
func (d *StandardEncodingDetector) initializeEncodings() {
	// UTF编码
	d.encodings["utf-8"] = unicode.UTF8
	d.encodings["utf-16"] = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	d.encodings["utf-16le"] = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	d.encodings["utf-16be"] = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	
	// 中文编码
	d.encodings["gb2312"] = simplifiedchinese.HZGB2312
	d.encodings["gbk"] = simplifiedchinese.GBK
	d.encodings["gb18030"] = simplifiedchinese.GB18030
	d.encodings["big5"] = traditionalchinese.Big5
	
	// 日文编码
	d.encodings["shift_jis"] = japanese.ShiftJIS
	d.encodings["euc-jp"] = japanese.EUCJP
	d.encodings["iso-2022-jp"] = japanese.ISO2022JP
	
	// 韩文编码
	d.encodings["euc-kr"] = korean.EUCKR
	
	// 西欧编码
	d.encodings["iso-8859-1"] = charmap.ISO8859_1
	d.encodings["windows-1252"] = charmap.Windows1252
}

// initializeDetectionRules 初始化检测规则
func (d *StandardEncodingDetector) initializeDetectionRules() {
	d.detectionRules = []DetectionRule{
		{
			Name:     "UTF-8 BOM",
			Encoding: "utf-8",
			Detector: d.detectUTF8BOM,
		},
		{
			Name:     "UTF-16 BOM",
			Encoding: "utf-16",
			Detector: d.detectUTF16BOM,
		},
		{
			Name:     "UTF-8",
			Encoding: "utf-8",
			Detector: d.detectUTF8,
		},
		{
			Name:     "GBK/GB2312",
			Encoding: "gbk",
			Detector: d.detectGBK,
		},
		{
			Name:     "Big5",
			Encoding: "big5",
			Detector: d.detectBig5,
		},
		{
			Name:     "Shift_JIS",
			Encoding: "shift_jis",
			Detector: d.detectShiftJIS,
		},
		{
			Name:     "EUC-KR",
			Encoding: "euc-kr",
			Detector: d.detectEUCKR,
		},
		{
			Name:     "ISO-8859-1",
			Encoding: "iso-8859-1",
			Detector: d.detectISO88591,
		},
	}
}

// DetectEncoding 检测编码
func (d *StandardEncodingDetector) DetectEncoding(data []byte) (string, float64, error) {
	if len(data) == 0 {
		return "utf-8", 1.0, nil
	}
	
	bestEncoding := "utf-8"
	bestConfidence := 0.0
	
	// 按优先级检测编码
	for _, rule := range d.detectionRules {
		if detected, confidence := rule.Detector(data); detected && confidence > bestConfidence {
			bestEncoding = rule.Encoding
			bestConfidence = confidence
		}
	}
	
	// 如果没有检测到高置信度的编码，尝试统计方法
	if bestConfidence < 0.8 {
		if encoding, confidence := d.detectByStatistics(data); confidence > bestConfidence {
			bestEncoding = encoding
			bestConfidence = confidence
		}
	}
	
	return bestEncoding, bestConfidence, nil
}

// DetectEncodingFromString 从字符串检测编码
func (d *StandardEncodingDetector) DetectEncodingFromString(text string) (string, float64, error) {
	return d.DetectEncoding([]byte(text))
}

// GetSupportedEncodings 获取支持的编码
func (d *StandardEncodingDetector) GetSupportedEncodings() []string {
	encodings := make([]string, 0, len(d.encodings))
	for encoding := range d.encodings {
		encodings = append(encodings, encoding)
	}
	return encodings
}

// 编码检测方法

// detectUTF8BOM 检测UTF-8 BOM
func (d *StandardEncodingDetector) detectUTF8BOM(data []byte) (bool, float64) {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return true, 1.0
	}
	return false, 0.0
}

// detectUTF16BOM 检测UTF-16 BOM
func (d *StandardEncodingDetector) detectUTF16BOM(data []byte) (bool, float64) {
	if len(data) >= 2 {
		if (data[0] == 0xFF && data[1] == 0xFE) || (data[0] == 0xFE && data[1] == 0xFF) {
			return true, 1.0
		}
	}
	return false, 0.0
}

// detectUTF8 检测UTF-8
func (d *StandardEncodingDetector) detectUTF8(data []byte) (bool, float64) {
	if utf8.Valid(data) {
		// 计算非ASCII字符比例来确定置信度
		nonASCII := 0
		for _, b := range data {
			if b > 127 {
				nonASCII++
			}
		}
		
		confidence := 0.9
		if nonASCII > 0 {
			confidence = 0.95 // 包含非ASCII字符的有效UTF-8更可能是UTF-8
		}
		
		return true, confidence
	}
	return false, 0.0
}

// detectGBK 检测GBK编码
func (d *StandardEncodingDetector) detectGBK(data []byte) (bool, float64) {
	gbkCount := 0
	totalBytes := len(data)
	
	for i := 0; i < totalBytes-1; i++ {
		b1, b2 := data[i], data[i+1]
		
		// GBK第一字节范围：0x81-0xFE
		// GBK第二字节范围：0x40-0x7E, 0x80-0xFE
		if (b1 >= 0x81 && b1 <= 0xFE) &&
			((b2 >= 0x40 && b2 <= 0x7E) || (b2 >= 0x80 && b2 <= 0xFE)) {
			gbkCount++
			i++ // 跳过下一个字节
		}
	}
	
	if gbkCount > 0 {
		confidence := float64(gbkCount*2) / float64(totalBytes)
		if confidence > 0.3 {
			return true, confidence
		}
	}
	
	return false, 0.0
}

// detectBig5 检测Big5编码
func (d *StandardEncodingDetector) detectBig5(data []byte) (bool, float64) {
	big5Count := 0
	totalBytes := len(data)
	
	for i := 0; i < totalBytes-1; i++ {
		b1, b2 := data[i], data[i+1]
		
		// Big5第一字节范围：0xA1-0xFE
		// Big5第二字节范围：0x40-0x7E, 0xA1-0xFE
		if (b1 >= 0xA1 && b1 <= 0xFE) &&
			((b2 >= 0x40 && b2 <= 0x7E) || (b2 >= 0xA1 && b2 <= 0xFE)) {
			big5Count++
			i++ // 跳过下一个字节
		}
	}
	
	if big5Count > 0 {
		confidence := float64(big5Count*2) / float64(totalBytes)
		if confidence > 0.3 {
			return true, confidence
		}
	}
	
	return false, 0.0
}

// detectShiftJIS 检测Shift_JIS编码
func (d *StandardEncodingDetector) detectShiftJIS(data []byte) (bool, float64) {
	sjisCount := 0
	totalBytes := len(data)
	
	for i := 0; i < totalBytes-1; i++ {
		b1, b2 := data[i], data[i+1]
		
		// Shift_JIS第一字节范围：0x81-0x9F, 0xE0-0xFC
		// Shift_JIS第二字节范围：0x40-0x7E, 0x80-0xFC
		if ((b1 >= 0x81 && b1 <= 0x9F) || (b1 >= 0xE0 && b1 <= 0xFC)) &&
			((b2 >= 0x40 && b2 <= 0x7E) || (b2 >= 0x80 && b2 <= 0xFC)) {
			sjisCount++
			i++ // 跳过下一个字节
		}
	}
	
	if sjisCount > 0 {
		confidence := float64(sjisCount*2) / float64(totalBytes)
		if confidence > 0.3 {
			return true, confidence
		}
	}
	
	return false, 0.0
}

// detectEUCKR 检测EUC-KR编码
func (d *StandardEncodingDetector) detectEUCKR(data []byte) (bool, float64) {
	euckrCount := 0
	totalBytes := len(data)
	
	for i := 0; i < totalBytes-1; i++ {
		b1, b2 := data[i], data[i+1]
		
		// EUC-KR第一字节范围：0xA1-0xFE
		// EUC-KR第二字节范围：0xA1-0xFE
		if (b1 >= 0xA1 && b1 <= 0xFE) && (b2 >= 0xA1 && b2 <= 0xFE) {
			euckrCount++
			i++ // 跳过下一个字节
		}
	}
	
	if euckrCount > 0 {
		confidence := float64(euckrCount*2) / float64(totalBytes)
		if confidence > 0.3 {
			return true, confidence
		}
	}
	
	return false, 0.0
}

// detectISO88591 检测ISO-8859-1编码
func (d *StandardEncodingDetector) detectISO88591(data []byte) (bool, float64) {
	// ISO-8859-1可以表示任何字节值，所以置信度较低
	for _, b := range data {
		if b > 127 {
			return true, 0.3 // 低置信度
		}
	}
	return true, 0.1 // 纯ASCII也可能是ISO-8859-1
}

// detectByStatistics 通过统计方法检测编码
func (d *StandardEncodingDetector) detectByStatistics(data []byte) (string, float64) {
	// 统计字节分布
	byteFreq := make([]int, 256)
	for _, b := range data {
		byteFreq[b]++
	}
	
	// 根据字节分布特征判断编码
	highByteCount := 0
	for i := 128; i < 256; i++ {
		highByteCount += byteFreq[i]
	}
	
	if highByteCount == 0 {
		return "ascii", 0.9
	}
	
	// 检查是否符合中文编码特征
	if d.hasChineseCharacteristics(byteFreq) {
		return "gbk", 0.6
	}
	
	return "utf-8", 0.5
}

// hasChineseCharacteristics 检查是否具有中文字符特征
func (d *StandardEncodingDetector) hasChineseCharacteristics(byteFreq []int) bool {
	// 检查GBK编码的第一字节范围
	gbkFirstByteCount := 0
	for i := 0x81; i <= 0xFE; i++ {
		gbkFirstByteCount += byteFreq[i]
	}
	
	// 如果高字节中有相当比例在GBK范围内，可能是中文
	totalHighBytes := 0
	for i := 128; i < 256; i++ {
		totalHighBytes += byteFreq[i]
	}
	
	if totalHighBytes > 0 && float64(gbkFirstByteCount)/float64(totalHighBytes) > 0.5 {
		return true
	}
	
	return false
}
