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

func charDet(b []byte) (string, error) {
	d := chardet.NewTextDetector()
	res, err := d.DetectBest(b)
	if err != nil {
		return "", err
	}
	return res.Charset, nil
}

/*
Only detect Character encoding
*/
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

// 試したいこと
// https://dev.classmethod.jp/articles/golang-iconv/

// https://qiita.com/KemoKemo/items/d135ddc93e6f87008521
func getFileNameWithoutExt(path string) string {
	// Fixed with a nice method given by mattn-san
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}
func main() {
	// utf-8ファイルを開く
	if len(os.Args) < 2 {
		panic("ファイルを指定してください")
	}
	filepath := filepath.Base(os.Args[1])
	// filepath := "DMR-08 エピソード2 グレイト・ミラクル 〜セブン・ヒーローVer．〜sjis.csv"
	srcFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()
	if err := os.Mkdir("convert", 0777); err != nil {
		fmt.Println(err)
	}
	filepath = "convert/" + filepath
	// 書き込み先ファイルを用意
	dstFile, err := os.Create(filepath)
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
	log.Println("done")
}
