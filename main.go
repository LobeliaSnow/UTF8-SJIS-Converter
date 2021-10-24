package main

// https://dev.classmethod.jp/articles/golang-iconv/
import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// 文字エンコードを取得
// https://moxtsuan.hatenablog.com/entry/nkf-go
func charDet(b []byte) (string, error) {
	d := chardet.NewTextDetector()
	res, err := d.DetectBest(b)
	if err != nil {
		return "", err
	}
	return res.Charset, nil
}
func Guess(file *os.File) (string, error) {
	input, err := ioutil.ReadAll(file)
	file.Seek(0, 0)
	if err != nil {
		return "", err
	}
	det, err := charDet(input)
	return det, err
}
func IsUTF8(fp *os.File) bool {
	charset, _ := Guess(fp)
	return charset == "UTF-8"
}

// utf8-sjis間で対応していない文字を置き換え
// https://teratail.com/questions/106106
const (
	NO_BREAK_SPACE = "\u00A0"
	WAVE_DASH      = "\u301C"
)

type runeWriter struct {
	w io.Writer
}

func (rw *runeWriter) Write(b []byte) (int, error) {
	var err error
	l := 0
loop:
	for len(b) > 0 {
		_, n := utf8.DecodeRune(b)
		if n == 0 {
			break loop
		}
		_, err = rw.w.Write(b[:n])
		if err != nil {
			_, err = rw.w.Write([]byte{'?'})
			if err != nil {
				break loop
			}
		}
		l += n
		b = b[n:]
	}
	return l, err
}

func ParseArgs() (string, string) {
	argsCount := len(os.Args)
	if argsCount < 2 {
		panic("引数を指定してください")
	}
	var inputPath string
	outputPath := "convert"
	// 引数パース
	for i := 1; i < argsCount; i++ {
		if i+1 >= argsCount {
			panic("引数の指定が不正です")
		}
		i += 1
		switch os.Args[i-1] {
		case "-i":
			inputPath = os.Args[i]
		case "-o":
			outputPath = os.Args[i]
		default:
			panic("引数の形式が違います")
		}
	}
	return inputPath, outputPath
}
func ConvertEncode(input_path, output_path string, output_is_dir bool) {
	srcFile, err := os.Open(input_path)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()
	var outputPath string
	if output_is_dir {
		if err := os.Mkdir(output_path, 0777); err != nil {
			fmt.Println(err)
		}
		outputPath = output_path + "/" + filepath.Base(input_path)
	} else {
		outputPath = output_path
	}
	// 書き込み先ファイルを用意
	dstFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()
	var tee io.Reader
	if IsUTF8(srcFile) {
		writer := (&runeWriter{transform.NewWriter(dstFile, japanese.ShiftJIS.NewEncoder())})
		tee = io.TeeReader(srcFile, writer)
	} else {
		reader := transform.NewReader(srcFile, japanese.ShiftJIS.NewDecoder())
		tee = io.TeeReader(reader, dstFile)
	}
	// 書き込み
	s := bufio.NewScanner(tee)
	for s.Scan() {
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

}
func TraverseDirectory(directory, output_path string, output_is_dir bool, work func(string, string, bool)) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if file.IsDir() {
			TraverseDirectory(directory+"/"+file.Name(), output_path, output_is_dir, work)
		} else {
			work(directory+"/"+file.Name(), output_path, output_is_dir)
		}
	}

}
func main() {
	inputPath, outputPath := ParseArgs()
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		panic("引数のパスを見直してください")
	}
	isInputDir := inputInfo.IsDir()
	if isInputDir && filepath.Ext(outputPath) != "" {
		panic("引数のディレクトリ関係が不正です")
	}
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		os.Mkdir(outputPath, 0777)
		outputInfo, err = os.Stat(outputPath)
		if err != nil {
			panic("引数のパスを見直してください")
		}
	}
	isOutputDir := outputInfo.IsDir()
	TraverseDirectory(inputPath, outputPath, isOutputDir, ConvertEncode)
	log.Println("done")
}
