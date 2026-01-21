package xhs

import (
	"encoding/json"
	"math/rand"
	"strings"
)

var base64Chars = []rune("ZmserbBoHQtNP+wOcza/LpngG8yJq42KWYj0DSfdikx3VT16IlUAFM97hECvuRX5")

var crc32Table = []int32{
	0, 1996959894, -301047508, -1727442502, 124634137, 1886057615, -379345611,
	-1637575261, 249268274, 2044508324, -522852066, -1747789432, 162941995,
	2125561021, -407360249, -1866523247, 498536548, 1789927666, -205950648,
	-2067906082, 450548861, 1843258603, -187386543, -2083289657, 325883990,
	1684777152, -43845254, -1973040660, 335633487, 1661365465, -99664541,
	-1928851979, 997073096, 1281953886, -715111964, -1570279054, 1006888145,
	1258607687, -770865667, -1526024853, 901097722, 1119000684, -608450090,
	-1396901568, 853044451, 1172266101, -589951537, -1412350631, 651767980,
	1373503546, -925412992, -1076862698, 565507253, 1454621731, -809855591,
	-1195530993, 671266974, 1594198024, -972236366, -1324619484, 795835527,
	1483230225, -1050599021, -1234817731, 1994146192, 31158534, -1731059524,
	-271249366, 1907459465, 112637215, -1614814043, -390540237, 2013776290,
	251722036, -1777751922, -519137256, 2137656763, 141376813, -1855689577,
	-429696000, 1802195444, 476864866, -2056965928, -228458418, 1812370925,
	453092731, -2113342271, -183516073, 1706088902, 314042704, -1950435094,
	-54949764, 1658658271, 366619977, -1932296973, -69972891, 1303535960,
	984961486, -1547960204, -725929758, 1256170817, 1037604311, -1529756563,
	-740887301, 1131014506, 879679996, -1385723834, -631195440, 1141124467,
	855842277, -1442165665, -586318647, 1342533948, 654459306, -1106571248,
	-921952122, 1466479909, 544179635, -1184443383, -832445281, 1591671054,
	702138776, -1328506846, -942167884, 1504918807, 783551873, -1212326853,
	-1061524307, -306674912, -1698712650, 62317068, 1957810842, -355121351,
	-1647151185, 81470997, 1943803523, -480048366, -1805370492, 225274430,
	2053790376, -468791541, -1828061283, 167816743, 2097651377, -267414716,
	-2029476910, 503444072, 1762050814, -144550051, -2140837941, 426522225,
	1852507879, -19653770, -1982649376, 282753626, 1742555852, -105259153,
	-1900089351, 397917763, 1622183637, -690576408, -1580100738, 953729732,
	1340076626, -776247311, -1497606297, 1068828381, 1219638859, -670225446,
	-1358292148, 906185462, 1090812512, -547295293, -1469587627, 829329135,
	1181335161, -882789492, -1134132454, 628085408, 1382605366, -871598187,
	-1156888829, 570562233, 1426400815, -977650754, -1296233688, 733239954,
	1555261956, -1026031705, -1244606671, 752459403, 1541320221, -1687895376,
	-328994266, 1969922972, 40735498, -1677130071, -351390145, 1913087877,
	83908371, -1782625662, -491226604, 2075208622, 213261112, -1831694693,
	-438977011, 2094854071, 198958881, -2032938284, -237706686, 1759359992,
	534414190, -2118248755, -155638181, 1873836001, 414664567, -2012718362,
	-15766928, 1711684554, 285281116, -1889165569, -127750551, 1634467795,
	376229701, -1609899399, -686959890, 1308918612, 956543938, -1486412191,
	-799009033, 1231636301, 1047427035, -1362007478, -640263460, 1088359270,
	936918000, -1447252397, -558129467, 1202900863, 817233897, -1111625188,
	-893730166, 1404277552, 615818150, -1160759803, -841546093, 1423857449,
	601450431, -1285129682, -1000256840, 1567103746, 711928724, -1274298825,
	-1022587231, 1510334235, 755167117,
}

// rightShiftUnsigned simulates JavaScript's >>> operator
func rightShiftUnsigned(num int32, bit uint) int32 {
	return int32(uint32(num) >> bit)
}

func mrc(e string) int32 {
	o := int32(-1)
	length := len(e)
	if length > 57 {
		length = 57
	}
	for n := 0; n < length; n++ {
		o = crc32Table[(o&255)^int32(e[n])] ^ rightShiftUnsigned(o, 8)
	}
	return o ^ -1 ^ -306674912
}

func tripletToBase64(e int) string {
	return string(base64Chars[(e>>18)&63]) +
		string(base64Chars[(e>>12)&63]) +
		string(base64Chars[(e>>6)&63]) +
		string(base64Chars[e&63])
}

func encodeChunk(data []int, start, end int) string {
	var sb strings.Builder
	for i := start; i < end; i += 3 {
		c := ((data[i] << 16) & 0xFF0000) + ((data[i+1] << 8) & 0xFF00) + (data[i+2] & 0xFF)
		sb.WriteString(tripletToBase64(c))
	}
	return sb.String()
}

func encodeUTF8(s string) []int {
	b := []byte(s)
	res := make([]int, len(b))
	for i, v := range b {
		res[i] = int(v)
	}
	return res
}

func b64Encode(data []int) string {
	length := len(data)
	remainder := length % 3
	var chunks []string

	mainLength := length - remainder
	for i := 0; i < mainLength; i += 16383 {
		end := i + 16383
		if end > mainLength {
			end = mainLength
		}
		chunks = append(chunks, encodeChunk(data, i, end))
	}

	if remainder == 1 {
		a := data[length-1]
		chunks = append(chunks, string(base64Chars[a>>2])+string(base64Chars[(a<<4)&63])+"==")
	} else if remainder == 2 {
		a := (data[length-2] << 8) + data[length-1]
		chunks = append(chunks, string(base64Chars[a>>10])+string(base64Chars[(a>>4)&63])+string(base64Chars[(a<<2)&63])+"=")
	}

	return strings.Join(chunks, "")
}

func getTraceId() string {
	const chars = "abcdef0123456789"
	result := make([]byte, 16)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func buildXsCommon(a1, b1, xs, xt string) string {
	payload := map[string]interface{}{
		"s0":  3,
		"s1":  "",
		"x0":  "1",
		"x1":  "4.2.2",
		"x2":  "Mac OS",
		"x3":  "xhs-pc-web",
		"x4":  "4.74.0",
		"x5":  a1,
		"x6":  xt,
		"x7":  xs,
		"x8":  b1,
		"x9":  mrc(xt + xs + b1),
		"x10": 154,
		"x11": "normal",
	}
	jsonBytes, _ := json.Marshal(payload)
	return b64Encode(encodeUTF8(string(jsonBytes)))
}

func buildXsPayload(x3Value string, dataType string) string {
	s := map[string]string{
		"x0": "4.2.1",
		"x1": "xhs-pc-web",
		"x2": "Mac OS",
		"x3": x3Value,
		"x4": dataType,
	}
	jsonBytes, _ := json.Marshal(s)
	return "XYS_" + b64Encode(encodeUTF8(string(jsonBytes)))
}

func buildSignString(uri string, data interface{}, method string) string {
	if strings.ToUpper(method) == "POST" {
		c := uri
		if data != nil {
			if strData, ok := data.(string); ok {
				c += strData
			} else {
				jsonBytes, _ := json.Marshal(data)
				c += string(jsonBytes)
			}
		}
		return c
	} else {
		if data == nil {
			return uri
		}
		return uri 
	}
}
