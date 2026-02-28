package client

import (
	"context"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
	authv1 "github.com/xiaoxiao0301/listen-stream-v2/server/shared/proto/auth/v1"
	"google.golang.org/grpc"
)

// AuthClient auth-svc gRPC客户端
type AuthClient struct {
	client  authv1.AuthServiceClient
	address string
	log     logger.Logger
}

// NewAuthClient 创建auth客户端
func NewAuthClient(conn *grpc.ClientConn, address string, log logger.Logger) *AuthClient {
	return &AuthClient{
		client:  authv1.NewAuthServiceClient(conn),
		address: address,
		log:     log,
	}
}

// VerifyToken 验证Token
func (c *AuthClient) VerifyToken(ctx context.Context, accessToken, clientIP, deviceFingerprint string) (*authv1.VerifyTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &authv1.VerifyTokenRequest{
		AccessToken:       accessToken,
		ClientIp:          clientIP,
		DeviceFingerprint: deviceFingerprint,
	}

	resp, err := c.client.VerifyToken(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
		).Error("Failed to verify token via gRPC")
		return nil, fmt.Errorf("verify token failed: %w", err)
	}

	return resp, nil
}

// RefreshToken 刷新Token
func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken, deviceID string) (*authv1.RefreshTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &authv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
		DeviceId:     deviceID,
	}

	resp, err := c.client.RefreshToken(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
		).Error("Failed to refresh token via gRPC")
		return nil, fmt.Errorf("refresh token failed: %w", err)
	}

	return resp, nil
}

// RevokeToken 撤销Token
func (c *AuthClient) RevokeToken(ctx context.Context, accessToken string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &authv1.RevokeTokenRequest{
		AccessToken: accessToken,
	}

	_, err := c.client.RevokeToken(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
		).Error("Failed to revoke token via gRPC")
		return fmt.Errorf("revoke token failed: %w", err)
	}

	return nil
}

// RevokeDevice 撤销设备
func (c *AuthClient) RevokeDevice(ctx context.Context, userID, deviceID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &authv1.RevokeDeviceRequest{
		UserId:   userID,
		DeviceId: deviceID,
	}

	_, err := c.client.RevokeDevice(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
		).Error("Failed to revoke device via gRPC")
		return fmt.Errorf("revoke device failed: %w", err)
	}

	return nil
}

// GetUserDevices 获取用户设备列表
func (c *AuthClient) GetUserDevices(ctx context.Context, userID string) (*authv1.GetUserDevicesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &authv1.GetUserDevicesRequest{
		UserId: userID,
	}

	resp, err := c.client.GetUserDevices(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to get user devices via gRPC")
		return nil, fmt.Errorf("get user devices failed: %w", err)
	}

	return resp, nil
}

// ValidateTokenVersion 验证Token版本
func (c *AuthClient) ValidateTokenVersion(ctx context.Context, userID string, tokenVersion int32) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &authv1.ValidateTokenVersionRequest{
		UserId:       userID,
		TokenVersion: tokenVersion,
	}

	resp, err := c.client.ValidateTokenVersion(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to validate token version via gRPC")
		return false, fmt.Errorf("validate token version failed: %w", err)
	}

	return resp.Valid, nil
}
