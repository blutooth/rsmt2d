package rsmt2d

import(
    "math"
    "errors"
    "crypto/sha256"

    "github.com/NebulousLabs/merkletree"
)

type dataSquare struct {
    square [][][]byte
    width uint
    chunkSize uint
    rowRoots [][]byte
    columnRoots [][]byte
}

func newDataSquare(data [][]byte) (*dataSquare, error) {
    width := int(math.Ceil(math.Sqrt(float64(len(data)))))
    if int(math.Pow(float64(width), 2)) != len(data) {
        return nil, errors.New("number of chunks must be a power of 2")
    }

    square := make([][][]byte, width)
    chunkSize := len(data[0])
    for i := 0; i < width; i++ {
        square[i] = data[i*width:i*width+width]

        for j := 0; j < width; j++ {
            if len(square[i][j]) != chunkSize {
                return nil, errors.New("all chunks must be of equal size")
            }
        }
    }

    return &dataSquare{
        square: square,
        width: uint(width),
        chunkSize: uint(chunkSize),
    }, nil
}

func (ds *dataSquare) extendSquare(extendedWidth uint, fillerChunk []byte) error {
    if (uint(len(fillerChunk)) != ds.chunkSize) {
        return errors.New("filler chunk size does not match data square chunk size")
    }

    newWidth := ds.width + extendedWidth
    newSquare := make([][][]byte, newWidth)

    fillerExtendedRow := make([][]byte, extendedWidth)
    for i := uint(0); i < extendedWidth; i++ {
        fillerExtendedRow[i] = fillerChunk
    }

    fillerRow := make([][]byte, newWidth)
    for i := uint(0); i < newWidth; i++ {
        fillerRow[i] = fillerChunk
    }

    row := make([][]byte, ds.width)
    for i := uint(0); i < ds.width; i++ {
        copy(row, ds.square[i])
        newSquare[i] = append(row, fillerExtendedRow...)
    }

    for i := ds.width; i < newWidth; i++ {
        newSquare[i] = make([][]byte, newWidth)
        copy(newSquare[i], fillerRow)
    }

    ds.square = newSquare
    ds.width = newWidth

    ds.resetRoots()

    return nil
}

func (ds *dataSquare) getRowSlice(x uint, y uint, length uint) [][]byte {
    return ds.square[x][y:y+length]
}

func (ds *dataSquare) getRow(x uint) [][]byte {
    return ds.getRowSlice(x, 0, ds.width)
}

func (ds *dataSquare) setRowSlice(x uint, y uint, newRow [][]byte) error {
    for i := uint(0); i < uint(len(newRow)); i++ {
        if len(newRow[i]) != int(ds.chunkSize) {
            return errors.New("invalid chunk size")
        }
    }

    for i := uint(0); i < uint(len(newRow)); i++ {
        ds.square[x][y+i] = newRow[i]
    }

    ds.resetRoots()

    return nil
}

func (ds *dataSquare) getColumnSlice(x uint, y uint, length uint) [][]byte {
    columnSlice := make([][]byte, length)
    for i := uint(0); i < length; i++ {
        columnSlice[i] = ds.square[x+i][y]
    }

    return columnSlice
}

func (ds *dataSquare) getColumn(y uint) [][]byte {
    return ds.getColumnSlice(0, y, ds.width)
}

func (ds *dataSquare) setColumnSlice(x uint, y uint, newColumn [][]byte) error {
    for i := uint(0); i < uint(len(newColumn)); i++ {
        if len(newColumn[i]) != int(ds.chunkSize) {
            return errors.New("invalid chunk size")
        }
    }

    for i := uint(0); i < uint(len(newColumn)); i++ {
        ds.square[x+i][y] = newColumn[i]
    }

    ds.resetRoots()

    return nil
}

func (ds *dataSquare) resetRoots() {
    ds.rowRoots = nil
    ds.columnRoots = nil
}

func (ds *dataSquare) computeRoots() {
    rowRoots := make([][]byte, ds.width)
    columnRoots := make([][]byte, ds.width)
    var rowTree *merkletree.Tree
    var columnTree *merkletree.Tree
    var rowData [][]byte
    var columnData [][]byte
    for i := uint(0); i < ds.width; i++ {
        rowTree = merkletree.New(sha256.New())
        columnTree = merkletree.New(sha256.New())
        rowData = ds.getRow(i)
        columnData = ds.getColumn(i)
        for j := uint(0); j < ds.width; j++ {
            rowTree.Push(rowData[j])
            columnTree.Push(columnData[j])
        }

        rowRoots[i] = rowTree.Root()
        columnRoots[i] = columnTree.Root()
    }

    ds.rowRoots = rowRoots
    ds.columnRoots = columnRoots
}

func (ds *dataSquare) RowRoots() [][]byte {
    if ds.rowRoots == nil {
        ds.computeRoots()
    }

    return ds.rowRoots
}

func (ds *dataSquare) ColumnRoots() [][]byte {
    if ds.columnRoots == nil {
        ds.computeRoots()
    }

    return ds.columnRoots
}

func (ds *dataSquare) computeRowProof(x uint, y uint) ([]byte, [][]byte, uint, uint) {
    tree := merkletree.New(sha256.New())
    tree.SetIndex(uint64(y))
    data := ds.getRow(x)

    for i := uint(0); i < ds.width; i++ {
        tree.Push(data[i])
    }

    merkleRoot, proof, proofIndex, numLeaves := tree.Prove()
    return merkleRoot, proof, uint(proofIndex), uint(numLeaves)
}

func (ds *dataSquare) computeColumnProof(x uint, y uint) ([]byte, [][]byte, uint, uint) {
    tree := merkletree.New(sha256.New())
    tree.SetIndex(uint64(x))
    data := ds.getColumn(y)

    for i := uint(0); i < ds.width; i++ {
        tree.Push(data[i])
    }

    merkleRoot, proof, proofIndex, numLeaves := tree.Prove()
    return merkleRoot, proof, uint(proofIndex), uint(numLeaves)
}
