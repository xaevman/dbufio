package dbufio

import (
    "crypto/sha1"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "math/rand"
    "os"
    "testing"
)

const (
    blockSize        = 1024 * 1024
    largeBufferSize  = 64 * 1024 * 1024
    mediumBufferSize = 64 * 1024
    smallBufferSize  = 64
    testFileSize     = blockSize * 512 // 25MB
)

var (
    testSizes = []int{
        32 * 1024,
        64 * 1024,
        13 * 1024 * 1024,
        32 * 1024 * 1024,
        64 * 1024 * 1024,
        256 * 1024 * 1024,
    }
)

var (
    err      error
    testFile *os.File
    testHash string
)

func TestMain(m *testing.M) {
    // must parse flags for test package to work
    flag.Parse()

    // generate a test file
    testFile, err = ioutil.TempFile("", "")
    if err != nil {
        panic(err)
    }
    defer testFile.Close()

    hash := sha1.New()

    // fill the temp file with random data
    fmt.Println("Filling file with random data...")
    buffer := make([]byte, blockSize)
    i := int64(0)
    for i < testFileSize {
        rand.Read(buffer)
        hash.Write(buffer)
        count, err := testFile.Write(buffer)
        if err != nil {
            panic(err)
        }

        i += int64(count)
    }

    if i != testFileSize {
        panic(fmt.Errorf("Test file size does not match (%d != %d)", i, testFileSize))
    }

    // record hash of file contents
    testHash = fmt.Sprintf("%X", hash.Sum(nil))
    fmt.Printf("Hash of file %s is %s\n", testFile.Name(), testHash)

    // run tests
    fmt.Println("Running tests...")
    result := m.Run()

    // close and delete test file
    testFile.Close()
    os.Remove(testFile.Name())

    // exit
    os.Exit(result)
}

func TestReads(t *testing.T) {
    for i := range testSizes {
        count, hash := testRead(i, t)
        if count != testFileSize {
            t.Fatalf("Size mismatch: %d != %d", count, testFileSize)
        }
        if hash != testHash {
            t.Fatalf("Hash mismatch: %s != %s", hash, testHash)
        }

    }
}

func testRead(i int, t *testing.T) (int64, string) {
    testFile.Seek(0, 0)

    hash := sha1.New()
    fmt.Printf("Test %d byte buffer\n", testSizes[i])

    reader := NewReader(testFile, testSizes[i])
    count, err := io.Copy(hash, reader)
    if err != nil && err != io.EOF {
        panic(err)
    }

    return count, fmt.Sprintf("%X", hash.Sum(nil))
}
