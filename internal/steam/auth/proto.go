package auth

import (
	"encoding/base64"
	"fmt"
	"trade-manager/internal/steam/auth/pb"

	"google.golang.org/protobuf/proto"
)

func SerializeCAuthentication_GetPasswordRSAPublicKey_Request(accountName string) (string, error) {
	req := &pb.CAuthentication_GetPasswordRSAPublicKey_Request{
		AccountName: proto.String(accountName),
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal RSAPublicKey request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(reqBytes), nil
}

func DeserializeCAuthentication_GetPasswordRSAPublicKey_Response(respB64 string) (*pb.CAuthentication_GetPasswordRSAPublicKey_Response, error) {
	raw, err := base64.StdEncoding.DecodeString(respB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	msg := new(pb.CAuthentication_GetPasswordRSAPublicKey_Response)
	if err := proto.Unmarshal(raw, msg); err != nil {
		return nil, fmt.Errorf("unmarshal RSAPublicKey request: %w", err)
	}
	return msg, nil
}

func SerializeCAuthentication_BeginAuthSessionViaCredentials_Request(
	accountName, encryptedPassword string,
	encryptionTimestamp uint64,
) (string, error) {

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"

	req := &pb.CAuthentication_BeginAuthSessionViaCredentials_Request{
		DeviceFriendlyName:  nil,
		AccountName:         proto.String(accountName),
		EncryptedPassword:   proto.String(encryptedPassword),
		EncryptionTimestamp: proto.Uint64(encryptionTimestamp),
		RememberLogin:       proto.Bool(true),
		PlatformType:        pb.EAuthTokenPlatformType_k_EAuthTokenPlatformType_Unknown.Enum(),
		Persistence:         pb.ESessionPersistence_k_ESessionPersistence_Persistent.Enum(),
		WebsiteId:           proto.String("Community"),
		DeviceDetails: &pb.CAuthentication_DeviceDetails{
			DeviceFriendlyName: proto.String(ua),
			PlatformType:       pb.EAuthTokenPlatformType_k_EAuthTokenPlatformType_WebBrowser.Enum(),
			OsType:             proto.Int32(0),
			GamingDeviceType:   proto.Uint32(0),
			ClientCount:        proto.Uint32(0),
			MachineId:          nil,
			AppType:            pb.EAuthTokenAppType_k_EAuthTokenAppType_Unknown.Enum(),
		},
		Language:  proto.Uint32(0),
		QosLevel:  proto.Int32(2),
		GuardData: nil,
	}

	reqBytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal BeginAuthSessionViaCredentials request: %w", err)
	}

	result := base64.StdEncoding.EncodeToString(reqBytes)
	return result, nil
}

func DeserializeBeginAuthSessionViaCredentialsRequest(enc string) (*pb.CAuthentication_BeginAuthSessionViaCredentials_Request, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(enc)
		if err != nil {
			return nil, fmt.Errorf("base64 decode failed (std and raw): %w", err)
		}
	}

	msg := new(pb.CAuthentication_BeginAuthSessionViaCredentials_Request)
	if err := proto.Unmarshal(raw, msg); err != nil {
		return nil, fmt.Errorf("proto unmarshal failed: %w", err)
	}

	return msg, nil
}

func DeserializeCAuthentication_BeginAuthSessionViaCredentials_Response(respB64 string) (*pb.CAuthentication_BeginAuthSessionViaCredentials_Response, error) {
	respBytes, err := base64.StdEncoding.DecodeString(respB64)
	if err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	resp := new(pb.CAuthentication_BeginAuthSessionViaCredentials_Response)
	if err := proto.Unmarshal(respBytes, resp); err != nil {
		return nil, fmt.Errorf("unmarshal BeginAuthSessionViaCredentials failed: %w", err)
	}

	return resp, nil
}

func SerializeCAuthentication_UpdateAuthSessionWithSteamGuardCode_Request(
	сlientId, steamId uint64,
	code string,
) (string, error) {

	req := &pb.CAuthentication_UpdateAuthSessionWithSteamGuardCode_Request{
		ClientId: proto.Uint64(сlientId),
		Steamid:  proto.Uint64(steamId),
		Code:     proto.String(code),
		CodeType: pb.EAuthSessionGuardType_k_EAuthSessionGuardType_DeviceCode.Enum(),
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal UpdateAuthSessionWithSteamGuardCode failed: %w", err)
	}

	result := base64.StdEncoding.EncodeToString(reqBytes)
	return result, nil
}

func SerializePollAuthSessionStatus_Request(
	clientId uint64, requestId []byte,
	tokenToRevoke uint64,
) (string, error) {

	req := &pb.CAuthentication_PollAuthSessionStatus_Request{
		ClientId:      proto.Uint64(clientId),
		RequestId:     requestId,
		TokenToRevoke: proto.Uint64(tokenToRevoke),
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal PollAuthSessionStatus failed: %w", err)
	}

	result := base64.StdEncoding.EncodeToString(reqBytes)
	return result, nil
}

func DeserializePollAuthSessionStatus_Response(payload string) (*pb.CAuthentication_PollAuthSessionStatus_Response, error) {
	respBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	resp := new(pb.CAuthentication_PollAuthSessionStatus_Response)
	if err := proto.Unmarshal(respBytes, resp); err != nil {
		return nil, fmt.Errorf("Unmarshal PollAuthSessionStatus failed: %w", err)
	}
	fmt.Println("\n--- Parsed Response ---")
	fmt.Println("New_ClientId:", resp.GetNewClientId())
	fmt.Println("New_ChallengeUrl:", resp.GetNewChallengeUrl())
	fmt.Println("Refresh_Token:", resp.GetRefreshToken())
	fmt.Println("Access_Token:", resp.GetAccessToken())
	fmt.Println("HadRemote_Interaction:", resp.GetHadRemoteInteraction())
	fmt.Println("Account_Name:", resp.GetAccountName())
	fmt.Println("New_GuardData:", resp.GetNewGuardData())
	fmt.Println("Agreement_SessionUrl:", resp.GetAgreementSessionUrl())
	return resp, nil
}
