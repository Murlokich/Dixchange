package jwt_test

import (
	"testing"

	"github.com/Murlokich/Dixchange/internal/auth/adapter/jwt"
	"github.com/Murlokich/Dixchange/internal/config"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Test-only RSA keypairs, not used anywhere outside this test.
const (
	testIssuer   = "test-issuer"
	testAudience = "test-audience"

	testPrivateKeyB64 = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRQ2FXakVmYkxPd1ZCUGoKeDhON2toeS9HWndjUkRLSEd1OFVQblZOVWtnM1UwaTVwREZWQkxGcjFRQ20xdzJwK1kvVitIYU8zblBJa3U3VQpiTmtsMldLWnpUTlVodkxPV3Q1NmgwUEVJeEZ6c1Njeld0ekJ6ZzN2WloyQW5JRXQ1ZVdIUEZkbEx6a3ZUcHUvCkh3Sm5CWENCWnNZc3dEbnkwYVJBUURXVm9ETnFuWEVvcXM1K3JaclA0NHgzeU5IN3pDK0ozTFlUVVFraDlZUmkKKzJyVmVUWFl2MzBWN0EvbXhVSW40cEhINUp6VU9kcy9SZ2FCTnphbklqOFpFSnpjdHZ0dFoxaTF1S1h1SGVEdQpkYXBTVTFTNVRCRVVGR1ZJT245cVpOSnY2bHhENFpCUU5BRG1oYTNSbUxsZkdpeEcrYk5uWktUSm9uU1JueHRZCkR1YWxzc2JwQWdNQkFBRUNnZ0VBRk12ZUhudkk0aE1SOHZpeFQrN3B6SjJvdU80L1dKQzZjYUhRc0RCTDVwaGsKRGc4ZlRuTXBiSHZ3Y0lvOUh3K3UzNGR3Z2VOYStrVkJ4SUU5NkttWGJWMkdjdmNUczlYdWI5ZlZ1NUYyQW41MQo1NmR3WXZSZVVFa0NwWmR3bUR5SFptY3M1MGNyajBJcUhlQVJ0TXU0MHRhRUVQZDlWMGdEdDF4dlc4K1AzZWRPCi9CWTNkTE1aT3BmQWo1VWJFMzNCMXZpbG1UYllDcjVHNldRRWl0WTBudFF1NklNaW5KbmphUGpSMFBFYUpHek0KL0ZUVFBKT3liR2NYWEFRdFhreDU4UHZzcytibDdoTThBZGdCUE9NQzhsQVhwQnNzOUJ3UHZ6cWRvQXIreWNCNwpFNHVBR0ZlaDYyNmM5Vi9FUlBIR0hqb1FrSERPL2tCOGVRNDBHa0VpK1FLQmdRRFE0MVZWMTRNSEhKMnB2V1B6CjFBWEZxMXlXY2I1MGFsUjNQSUM3dlZiMm5iZVVsUHdwZUZXYkV5ZmhpM3ZVNGZ5TEVOQm8rUUZtTm9pd2laUDgKenpxVFFXd1Nva05HVm9seFI4YXk0dTFjR0I5emsxWDJvcmxnNVdxQ25mN0RSNENPbGJZOE1Ba1RIVUVydkRzYwo2TVg5QWZMNUd0bjlKSEkvL3locTNzMGY2d0tCZ1FDOUtocS9JcGsyZUlHeEhPc3lyYzlycFVHS3lqVG1iYzVZCjFFbnIxajM2cWRDM0JlUzhEWGpmTDBlQ1VjZGhOa2taNENEYTVZbWhXcFIyRHgrVUdCb0FPTXpIajBRbVZXTGIKZUdscGhhZ3hwY2hyZzhzWm1yOTAwZ2MzcmZPLzBFWXhyajFxbkN0NkZpVlQ0bWFqSmlGTSt5ZGV1UjlYMmpOVQpmVXQxVTlJVGV3S0JnSEpWL0wrZjF2aHRrVjczYTE1ckR6M0RMa3FxdDdUQS9yckZRekYvMGJucldqRFoyNnhYClVZUXdXQVA4ZFN0eEJ1cVhxWHZMMEJoQ2dVOGYvd0RCN1MzdEFWcXh0bUVwdk5Ga21ENFNPQm5RbW1hSW51VUEKN0xjODI2L2pDeStZVTZXNERLZWxJMUp6RU5qL2p0dWtEYnNPOWcrand1M3NwUnlpYlYwUjF2VkJBb0dBY21ZTwpPdDFmcnhOOXkzRGp4SWhjMG9ya2NsMExWUGUvUFBWVnB5eXYydkhyNldQRkN2MVN4aGEvdGs1aXQ2V3UyOWt4CnhXL2ZwaGN6aFpwcnpKSXUxMGVDZ0lWaC81VFA3azZKNFdrVmFNRUx6Z21zUUZYWHgzeGxKUVVzVGlwVTZTS28KdmRxb2FITVFMQVRWY1lGVHpreHRhTm1uM09kMjkzZTJ0TVAyWGhVQ2dZRUF5V2F2UkF6R09tMTV4cFkzNE9DQQpCOXl0TVorRkNKOVJmUEZ5cGl0ZDFsSTMxems3KzhJYkpQZmFuTGxGSjdZOHE2d0p0czU3OFM5dWhYSy9VYmVqCkt2TWlwYnlYQ2U0d1dBN0k1NEo4dndXTWhnR0ZCcmhxMlhIc3lUWis1TU1IZTBlVHRPZklTamdncnpualh5KzYKbTlZZEU4NlVRb3NRZXhISFJXK2lINWM9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K" //nolint:lll
	testPublicKeyB64  = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFtbG94SDJ5enNGUVQ0OGZEZTVJYwp2eG1jSEVReWh4cnZGRDUxVFZKSU4xTkl1YVF4VlFTeGE5VUFwdGNOcWZtUDFmaDJqdDV6eUpMdTFHelpKZGxpCm1jMHpWSWJ5emxyZWVvZER4Q01SYzdFbk0xcmN3YzRONzJXZGdKeUJMZVhsaHp4WFpTODVMMDZidng4Q1p3VncKZ1diR0xNQTU4dEdrUUVBMWxhQXphcDF4S0tyT2ZxMmF6K09NZDhqUis4d3ZpZHkyRTFFSklmV0VZdnRxMVhrMQoyTDk5RmV3UDVzVkNKK0tSeCtTYzFEbmJQMFlHZ1RjMnB5SS9HUkNjM0xiN2JXZFl0YmlsN2gzZzduV3FVbE5VCnVVd1JGQlJsU0RwL2FtVFNiK3BjUStHUVVEUUE1b1d0MFppNVh4b3NSdm16WjJTa3lhSjBrWjhiV0E3bXBiTEcKNlFJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     //nolint:lll

	testPrivateKey2B64 = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQytYYTFTb0pXZnU1ajQKMkZIOFF6THJidWIrK1hqc2tvQStwdGQyOE9QbE9UVFVMTUE0QUVQOFVwZEw1Z2s1WllPaUpVZ0lpMld3WmxkagpJUkM1SWhrVFQzczFOSUVpY2cxU1FDYWNqS25kMElkTmJOMTQybUhjU2NJQk5la2RYVzgrQUhwOE9NWTNTWFkyCmt1WDNQbzc0d2hIOHE2dUNxV0xySlJ0Z0hLK21WTTZYczJvdkdZbWRKYmE1V1FERGRwQkZFemlBa2g4cnFDNmEKU3l4ekZtdGZ0dXp3WHJtdWlISGUzQTFuajljcHdGUTZ2RHc0SktFRFlJbUszejZEU1NOQ21UR2hwRCtwR1hxbQpSekRIVmhIWGlxSTZNNHhraW1jWUpqTDRhMGFSRXpHem1teHJydEJGRkZaN3hGTFZUZXVaSUxNNTdQczM1bnNhCml5S0dORE14QWdNQkFBRUNnZ0VBQXQ1UGU4UytyR3I3dVpnczJCTzlJeTRISGdWdFRJZENsbThXSlEwcEtlMk0KUGJ3MDQwTWh3eDROZUkvTlNNZWVUYVdwS2JQTUF2WHYwQ2NSNVBBK0YzdmxtOTRnS2dBMDhpM1VSeHlaZktXbApQRVh0S21LRUFmazhpcDhnMVpmd1VxdFlMU0gvM3BwbFJSbytpOVN0KzlLVXljMWVacmdkclBSZ3JyaVA3djRtCmVUUnBFSVBjVVdSbDJWYUlNM0Y5bGRQcmdwejNla1dVSEI2OFhLdS9KRHJMTTZwcUFZOWdibjFHeVdOMEl3K1YKNUdyOEg0R3UrRzUrczdGTGNWMFlZMHF3Um93WFc0ZzVPcWlUWjVMbjVsNFlqOVJXMGFzVytyQmlmQnJheHU0SApjZGtOaXdPWlFsRFlwME5QU3owUUwydHBjOE1ZTUxoV2Rpd0ZNMmgwNFFLQmdRRHRUenJ3ZCtZTVlRRVo2VWZNCmx6MTBQck1IREVNazBaUEVCa1hWM1YwWkRQS3h3WkJDNlltcG9DcVVsU1R3ZlRybCtnNVh6bi9xWXBOZnVaNUkKWmdIQUp1NEJnRTlLTlZGZnozYjJQSzFrbGowWTY5KzRoSHpidDFRZWxrSEpGbmU3czMzVVNPd05rSnU2S0k5WApSUDVPZm0zL3krTm43SVJhREFmR2liWktHUUtCZ1FETlcvR1p3RXRBeERBM1Frc2lTcUphaklnSFViUzAwRWRqClJKcUpLVFFlM3NYdkd6UXIydytpajJTSkhIK0JERUMrZkpoK2lJc045bHdaa1BhR05nYkZZN2UrWkdGTUl1Q3EKT3MwWjk5a3BYRVBWU0dQOTNzZkZQd1ptc2V1UDhOOEdqeFZkSkNaZUpFNWN5R1NyRHBrZ3pZeC9CS1RmYmZ3UwpKZGwxeko4RTJRS0JnUURnRkE5SVh6N0hjNVZ4blJoaDlIRmY0WkIwMkR4U2VONjVtMTYzK2VWZHNSODlmOU96Cml3ZWxOd3JkOEJMSVRzZWR2c2RxWEIvbTl4dkhxT0x2NC9wRk1iMU5oNVhYTVk5Y0UrOTF4WWZJMGlCdzl0RHkKVDVBQ2ZCS2ZLeXlGTDRUeHBTUHNuc3JlYUN6VDFGYTloZlZIc3hRMUZKWHZYS01BMEJsTGZGKzNpUUtCZ1FDbQpSYWFWcll1QXNoaXo5VG91VnEyamtSdG1pYkk5b0ZIY3Z5UThvQjd5UzcrSmZNWStLQTA5dXNwVTliMEsrS2ZIClU4eUl6dmQvNDVQb28yWG5md0QyZTJtL2Z6a0VVMDBDcDc5L0loMHpqdC9oWmM2SU5BUWQ0WXpHUTVLT0ppSCsKQm9YdXR5RlZvaUhNbi9oYVErU2FHYk9DK0hpaDYwUm9VVHNFVlJ2LzZRS0JnRE5NSGFrQVFkYlhoQXAyc3h3YQpueVB0ZGo5LzJSR1JPa1JLSHhQNm45V3l5eDhjWGZFbXZkQXB6WGJ0cnVLWUkwWDVsOUVGZ3VhQ1AvTXAySXA0CnUvcW5GTTBtRWRIMTlzSGFCUytiK1BEOU1sdDg4RkFValZycjU2S2N3NHh3ZXdjbE9QbTl5b3hQZnpNZ1JhVncKSjZiY1RPTERUaDVwd3h2Sms4VWxDRjVSCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K" //nolint:lll
	testPublicKey2B64  = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF2bDJ0VXFDVm43dVkrTmhSL0VNeQo2MjdtL3ZsNDdKS0FQcWJYZHZEajVUazAxQ3pBT0FCRC9GS1hTK1lKT1dXRG9pVklDSXRsc0daWFl5RVF1U0laCkUwOTdOVFNCSW5JTlVrQW1uSXlwM2RDSFRXemRlTnBoM0VuQ0FUWHBIVjF2UGdCNmZEakdOMGwyTnBMbDl6Nk8KK01JUi9LdXJncWxpNnlVYllCeXZwbFRPbDdOcUx4bUpuU1cydVZrQXczYVFSUk00Z0pJZks2Z3Vta3NzY3hacgpYN2JzOEY2NXJvaHgzdHdOWjQvWEtjQlVPcnc4T0NTaEEyQ0ppdDgrZzBralFwa3hvYVEvcVJsNnBrY3d4MVlSCjE0cWlPak9NWklwbkdDWXkrR3RHa1JNeHM1cHNhNjdRUlJSV2U4UlMxVTNybVNDek9lejdOK1o3R29zaWhqUXoKTVFJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     //nolint:lll
)

var testJWTConfig = &config.JWTConfig{ //nolint:gochecknoglobals // acceptable in test
	Issuer:        testIssuer,
	Audience:      testAudience,
	PublicKeyB64:  testPublicKeyB64,
	PrivateKeyB64: testPrivateKeyB64,
}

var testJWTConfig2 = &config.JWTConfig{ //nolint:gochecknoglobals // acceptable in test
	Issuer:        testIssuer,
	Audience:      testAudience,
	PublicKeyB64:  testPublicKey2B64,
	PrivateKeyB64: testPrivateKey2B64,
}

func TestManager_IssueAndValidateToken(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e33")

	manager, err := jwt.NewManager(testJWTConfig)
	require.NoError(t, err)

	accessToken, err := manager.IssueToken(userID)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)

	token, err := manager.ValidateToken(accessToken)
	require.NoError(t, err)
	require.True(t, token.Valid)

	subject, err := token.Claims.GetSubject()
	require.NoError(t, err)
	require.Equal(t, userID.String(), subject)

	issuer, err := token.Claims.GetIssuer()
	require.NoError(t, err)
	require.Equal(t, testIssuer, issuer)

	audience, err := token.Claims.GetAudience()
	require.NoError(t, err)
	require.Equal(t, []string{testAudience}, []string(audience))
}

func TestManager_ValidateToken_WrongSigningKey(t *testing.T) {
	t.Parallel()

	issuingManager, err := jwt.NewManager(testJWTConfig)
	require.NoError(t, err)

	validatingManager, err := jwt.NewManager(testJWTConfig2)
	require.NoError(t, err)

	accessToken, err := issuingManager.IssueToken(uuid.New())
	require.NoError(t, err)

	_, err = validatingManager.ValidateToken(accessToken)
	require.Error(t, err)
	require.ErrorIs(t, err, jwtlib.ErrTokenSignatureInvalid)
	require.ErrorContains(t, err, "signature is invalid")
}
