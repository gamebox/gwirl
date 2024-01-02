package parser

type input struct {
	source_ string
	offset_ int
	length_ int
}

func (in *input) apply(length int) string {
	if length == 0 {
		return string(in.source_[in.offset_])
	}
	return in.source_[in.offset_ : in.offset_+length]
}

func (in *input) matches(str string) bool {
	i := 0
	l := len(str)
	for i < l {
		if in.source_[in.offset_+i] != str[i] {
			return false
		}
		i = i + 1
	}
	return true
}

func (in *input) advance(increment int) {
	in.offset_ = in.offset_ + increment
}

func (in *input) regress(decrement int) {
	in.offset_ = in.offset_ - decrement
}

func (in *input) regressTo(offset int) {
	in.offset_ = offset
}

func (in *input) isPastEOF(len int) bool {
	return (in.offset_ + (len - 1)) >= in.length_
}

func (in *input) isEOF() bool {
	return in.isPastEOF(1)
}

func (in *input) atEnd() bool {
	return in.isEOF()
}

func (in *input) pos() {

}

func (in *input) offset() int {
	return in.offset_
}

func (in *input) source() string {
	return in.source_
}

func (in *input) reset(source string) {
	in.offset_ = 0
	in.source_ = source
	in.length_ = len(source)
}
