package components

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

const bitsInWord = 32

// GetComponentName returns the component name for the specific performance profile
func GetComponentName(profileName string, prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, profileName)
}

// GetFirstKeyAndValue return the first key / value pair of a map
func GetFirstKeyAndValue(m map[string]string) (string, string) {
	for k, v := range m {
		return k, v
	}
	return "", ""
}

// SplitLabelKey returns the given label key splitted up in domain and role
func SplitLabelKey(s string) (domain, role string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Can't split %s", s)
	}
	return parts[0], parts[1], nil
}

// CPUListToHexMask converts a list of cpus into a cpu mask represented in hexdecimal
func CPUListToHexMask(cpulist string) (hexMask string, err error) {
	cpus, err := cpuset.Parse(cpulist)
	if err != nil {
		return "", err
	}

	reservedCPUs := cpus.ToSlice()
	currMask := big.NewInt(0)
	for _, cpu := range reservedCPUs {
		x := new(big.Int).Lsh(big.NewInt(1), uint(cpu))
		currMask.Or(currMask, x)
	}
	return fmt.Sprintf("%064x", currMask), nil
}

// CPUListToMaskList converts a list of cpus into a cpu mask represented
// in a list of hexadecimal mask devided by a delimiter ","
func CPUListToMaskList(cpulist string) (hexMask string, err error) {
	maskStr, err := CPUListToHexMask(cpulist)
	if err != nil {
		return "", nil
	}
	index := 0
	for index < (len(maskStr) - 8) {
		if maskStr[index:index+8] != "00000000" {
			break
		}
		index = index + 8
	}
	var b bytes.Buffer
	for index <= (len(maskStr) - 16) {
		b.WriteString(maskStr[index : index+8])
		b.WriteString(",")
		index = index + 8
	}
	b.WriteString(maskStr[index : index+8])
	trimmedCPUMaskList := b.String()
	return trimmedCPUMaskList, nil
}

// CPULists allows easy checks between reserved and isolated cpu set definitons
type CPULists struct {
	reserved cpuset.CPUSet
	isolated cpuset.CPUSet
}

// Intersect returns cpu ids found in both the provided cpuLists, if any
func (c *CPULists) Intersect() []int {
	commonSet := c.reserved.Intersection(c.isolated)
	return commonSet.ToSlice()
}

// CountIsolated returns how many isolated cpus where specified
func (c *CPULists) CountIsolated() int {
	return c.isolated.Size()
}

// NewCPULists parse text representations of reserved and isolated cpusets definiton and returns a CPULists object
func NewCPULists(reservedList, isolatedList string) (*CPULists, error) {
	var err error
	reserved, err := cpuset.Parse(reservedList)
	if err != nil {
		return nil, err
	}
	isolated, err := cpuset.Parse(isolatedList)
	if err != nil {
		return nil, err
	}
	return &CPULists{
		reserved: reserved,
		isolated: isolated,
	}, nil
}

// CPUMaskToCPUSet parses a CPUSet received in a Mask Format, see:
// https://man7.org/linux/man-pages/man7/cpuset.7.html#FORMATS
func CPUMaskToCPUSet(cpuMask string) (cpuset.CPUSet, error) {
	chunks := strings.Split(cpuMask, ",")

	// reverse the chunks order
	n := len(chunks)
	for i := 0; i < n/2; i++ {
		chunks[i], chunks[n-i-1] = chunks[n-i-1], chunks[i]
	}

	builder := cpuset.NewBuilder()
	for i, chunk := range chunks {
		mask, err := strconv.ParseUint(chunk, 16, bitsInWord)
		if err != nil {
			return cpuset.NewCPUSet(), fmt.Errorf("failed to parse the CPU mask %s: %v", cpuMask, err)
		}
		for j := 0; j < bitsInWord; j++ {
			if mask&1 == 1 {
				builder.Add(i*bitsInWord + j)
			}
			mask >>= 1
		}
	}

	return builder.Result(), nil
}
