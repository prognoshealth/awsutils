package lambdautils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckIfS3TestEvent(t *testing.T) {
	cases := []struct {
		message  string
		expected bool
	}{
		{"some strange message that won't unmarshal", false},
		{"{\"Service\":\"Amazon S3\",\"Event\":\"s3:TestEvent\",\"Time\":\"2018-08-15T19:15:27.958Z\",\"Bucket\":\"bname\",\"RequestId\":\"E3D11FAF78CE1E52\",\"HostId\":\"vG00zg9q52/1ZSixeQW1CEnKe/mM5xJVja6QlOfbewmrLN8vNzPFPSKYr1Rzut0wwXL44J/M2N8=\"}", true},
		{"{\"Service\":\"Amazon S3\",\"Event\":\"some other event\",\"Time\":\"2018-08-15T19:15:27.958Z\",\"Bucket\":\"bname\",\"RequestId\":\"E3D11FAF78CE1E52\",\"HostId\":\"vG00zg9q52/1ZSixeQA1CEnHe/mM5xJVja6QlOfbewmrLN8vNzPFPSKYr1Rzut0wwXL44J/M2N8=\"}", false},
	}

	for _, c := range cases {
		actual := CheckIfS3TestEvent(c.message)
		assert.Equal(t, c.expected, actual)
	}
}
