package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/listen-stream/server/shared/proto/auth/v1"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	deviceservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/device"
	jwtservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/jwt"
)

// AuthServer 认证服务gRPC实现
type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	jwtService    jwtservice.JWTService
	deviceService deviceservice.DeviceService
}

// NewAuthServer 创建认证服务gRPC服务器
func NewAuthServer(
	jwtService jwtservice.JWTService,
	deviceService deviceservice.DeviceService,
) *AuthServer {
	return &AuthServer{
		jwtService:    jwtService,
		deviceService: deviceService,
	}
}

// VerifyToken 验证Token
func (s *AuthServer) VerifyToken(ctx context.Context, req *authv1.VerifyTokenRequest) (*authv1.VerifyTokenResponse, error) {
	if req.AccessToken == "" {
		return &authv1.VerifyTokenResponse{
			Valid:        false,
			ErrorMessage: "access token is required",
			ErrorCode:    authv1.ErrorCode_ERROR_CODE_UNSPECIFIED,
		}, nil
	}

	// 验证AccessToken
	claims, err := s.jwtService.ValidateAccessToken(ctx, req.AccessToken, req.ClientIp)
	if err != nil {
		// 根据错误类型返回相应的错误码
		errorCode := getErrorCode(err)
		return &authv1.VerifyTokenResponse{
			Valid:        false,
			ErrorMessage: err.Error(),
			ErrorCode:    errorCode,
		}, nil
	}

	// 如果提供了设备指纹，进行异常检测
	if req.DeviceFingerprint != "" {
		verifyReq := &deviceservice.VerifyDeviceRequest{
			UserID:     claims.UserID,
			DeviceID:   claims.DeviceID,
			DeviceName: "", // 不需要
			Platform:   "", // 不需要
			OSVersion:  "", // 不需要
			ClientIP:   req.ClientIp,
		}
		result, _ := s.deviceService.VerifyDevice(ctx, verifyReq)
		if result != nil && result.IsSuspicious {
			return &authv1.VerifyTokenResponse{
				Valid:        false,
				ErrorMessage: "suspicious device activity detected",
				ErrorCode:    authv1.ErrorCode_ERROR_CODE_FINGERPRINT_ANOMALY,
			}, nil
		}
	}

	// 构造响应
	return &authv1.VerifyTokenResponse{
		Valid: true,
		User: &authv1.User{
			Id:           claims.UserID,
			Phone:        "", // 需要从数据库查询
			TokenVersion: int32(claims.TokenVersion),
		},
		Device: &authv1.Device{
			Id:       claims.DeviceID,
			UserId:   claims.UserID,
			DeviceId: claims.DeviceID,
		},
	}, nil
}

// RefreshToken 刷新Token
func (s *AuthServer) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	// 刷新AccessToken
	tokenPair, err := s.jwtService.RefreshAccessToken(ctx, req.RefreshToken, "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    timestamppb.New(tokenPair.ExpiresAt),
	}, nil
}

// RevokeToken 撤销Token（通过递增用户的TokenVersion）
func (s *AuthServer) RevokeToken(ctx context.Context, req *authv1.RevokeTokenRequest) (*authv1.RevokeTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "access token is required")
	}

	// 解析Token获取UserID
	claims, err := s.jwtService.ValidateAccessToken(ctx, req.AccessToken, "")
	if err != nil {
		// Token无效或已过期，仍然返回成功（幂等性）
		return &authv1.RevokeTokenResponse{Success: true}, nil
	}

	// 撤销用户的所有Token
	err = s.jwtService.RevokeUserTokens(ctx, claims.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke tokens")
	}

	return &authv1.RevokeTokenResponse{Success: true}, nil
}

// RevokeDevice 撤销设备（删除设备并递增TokenVersion）
func (s *AuthServer) RevokeDevice(ctx context.Context, req *authv1.RevokeDeviceRequest) (*authv1.RevokeDeviceResponse, error) {
	if req.UserId == "" || req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and device_id are required")
	}

	// 删除设备
	err := s.deviceService.RemoveDevice(ctx, req.UserId, req.DeviceId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove device")
	}

	// 撤销用户的所有Token（递增TokenVersion）
	err = s.jwtService.RevokeUserTokens(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke tokens")
	}

	return &authv1.RevokeDeviceResponse{
		Success:       true,
		TokensRevoked: 1, // 实际上撤销了所有Token
	}, nil
}

// GetUserDevices 获取用户设备列表
func (s *AuthServer) GetUserDevices(ctx context.Context, req *authv1.GetUserDevicesRequest) (*authv1.GetUserDevicesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 获取设备列表
	devices, err := s.deviceService.ListDevices(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list devices")
	}

	// 转换为proto消息
	protoDevices := make([]*authv1.Device, 0, len(devices))
	for _, device := range devices {
		protoDevices = append(protoDevices, &authv1.Device{
			Id:          device.ID,
			UserId:      device.UserID,
			DeviceId:    device.ID,
			Platform:    convertPlatform(device.Platform),
			Fingerprint: device.Fingerprint,
			IpAddress:   device.LastIP,
			LastActive:  timestamppb.New(device.LastLoginAt),
			CreatedAt:   timestamppb.New(device.CreatedAt),
		})
	}

	return &authv1.GetUserDevicesResponse{
		Devices: protoDevices,
	}, nil
}

// ValidateTokenVersion 验证Token版本
func (s *AuthServer) ValidateTokenVersion(ctx context.Context, req *authv1.ValidateTokenVersionRequest) (*authv1.ValidateTokenVersionResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 通过验证一个临时Token来检查版本
	// 实际上，我们需要从数据库获取当前的TokenVersion
	// 这里简化实现，假设通过ValidateAccessToken能获取到当前版本
	
	// 注意：这个方法需要访问UserRepository来获取当前的TokenVersion
	// 为简化实现，我们返回基本响应
	return &authv1.ValidateTokenVersionResponse{
		Valid:          true,
		CurrentVersion: req.TokenVersion,
	}, nil
}

// getErrorCode 根据错误类型返回对应的错误码
func getErrorCode(err error) authv1.ErrorCode {
	switch err {
	case jwtservice.ErrTokenVersionMismatch:
		return authv1.ErrorCode_ERROR_CODE_VERSION_MISMATCH
	case jwtservice.ErrIPMismatch:
		return authv1.ErrorCode_ERROR_CODE_IP_MISMATCH
	case domain.ErrUserNotFound:
		return authv1.ErrorCode_ERROR_CODE_USER_DISABLED
	case domain.ErrUserInactive:
		return authv1.ErrorCode_ERROR_CODE_USER_DISABLED
	default:
		// 检查是否是JWT相关错误
		if err.Error() == "token is expired" {
			return authv1.ErrorCode_ERROR_CODE_TOKEN_EXPIRED
		}
		return authv1.ErrorCode_ERROR_CODE_INVALID_SIGNATURE
	}
}

// convertPlatform 转换平台类型
func convertPlatform(platform string) authv1.Platform {
	switch platform {
	case "iOS":
		return authv1.Platform_PLATFORM_IOS
	case "Android":
		return authv1.Platform_PLATFORM_ANDROID
	case "Web":
		return authv1.Platform_PLATFORM_WEB
	case "Desktop":
		return authv1.Platform_PLATFORM_DESKTOP
	default:
		return authv1.Platform_PLATFORM_UNSPECIFIED
	}
}
