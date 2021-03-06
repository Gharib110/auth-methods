package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log"
	"os"
	"time"
)

// KeyGenerator use for creating the key from bytes storage
func (conf *AppConfig) KeyGenerator() {
	for i := 0; i < conf.HmacConf.HashAlgorithm.Size(); i += 1 {
		j := conf.Rnd.Intn(2000)
		conf.Key = append(conf.Key, randomBytes[j])
	}

	return
}

// UUIDKeyGenerator use random bytes that generates by KeyGenerator function then create uuid based on that
func (conf *AppConfig) UUIDKeyGenerator() {
	conf.KeyGenerator()
	uid := uuid.FromBytesOrNil(conf.Key)

	conf.Key = uid.Bytes()
}

// EncodingBase64 use for encoding our msg to base64 for using alongside of another functionalities
func (conf *AppConfig) EncodingBase64(encoding *base64.Encoding, msg string) string {
	conf.Base64Encoder = encoding
	encodedStr := conf.Base64Encoder.EncodeToString([]byte(msg))

	return encodedStr
}

// DecodingBase64 use for decoding our msg to base64 for using alongside of another functionalities
func (conf *AppConfig) DecodingBase64(encodedStr string) ([]byte, error) {
	if conf.Base64Encoder == nil {
		return nil, errors.New("encoder is not provided")
	}

	result, err := conf.Base64Encoder.DecodeString(encodedStr)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return result, nil
}

// GenerateHash with bcrypt package
func (conf *AppConfig) GenerateHash(password string) error {
	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), conf.Cost)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	conf.HashPassword = hashPass

	return nil
}

// ComparePassAndHash for comparing hashPass and password
func (conf *AppConfig) ComparePassAndHash(password string) error {
	err := bcrypt.CompareHashAndPassword(conf.HashPassword, []byte(password))
	if err != nil {
		log.Println(err.Error())
		return err
	}

	fmt.Println("Hash and Password matches !")
	return nil
}

// HmacSigToken use for sign specific token and then return signed token
func (conf *AppConfig) HmacSigToken(token []byte) ([]byte, error) {
	conf.HmacConf.HmacSigner = hmac.New(conf.HmacConf.HashMethod, conf.Key)

	_, err := conf.HmacConf.HmacSigner.Write(token)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return conf.HmacConf.HmacSigner.Sum(nil), nil
}

// CheckSignMsg use for checking the sign of received token with existing token
func (conf *AppConfig) CheckSignMsg(token, oldSign []byte) (bool, error) {
	newSign, err := conf.HmacSigToken(token)
	if err != nil {
		log.Println(err.Error())
		return false, err
	}

	identifier := hmac.Equal(oldSign, newSign)

	return identifier, nil
}

// Valid check the validation for our JWT token
func (u *UserClaims) Valid() error {
	checkExpire := u.VerifyExpiresAt(time.Now().UnixNano(), true)

	if !checkExpire {
		return fmt.Errorf("token has expired")
	}

	return nil
}

// CreateSignedToken use for creating JWT token
func (conf *AppConfig) CreateSignedToken(uc *UserClaims) (string, error) {
	token := jwt.NewWithClaims(conf.JwtConf.JwtKeyMethod, uc)
	signedToken, err := token.SignedString(conf.Key)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}

	return signedToken, nil
}

// ParseSignedToken use for parsing the signed token to our claims
func (conf *AppConfig) ParseSignedToken(signedToken string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(signedToken, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
			return nil, fmt.Errorf("invalid signing signing algorithm")
		}

		return conf.Key, nil
	})

	if err != nil {
		return nil, errors.New("error in parsing token : " + err.Error())
	}

	if !token.Valid {
		return nil, errors.New("error in parsing token : token is invalid")
	}

	return token.Claims.(*UserClaims), nil
}

// EncryptSHA256File use for encrypting a file with sha256 method
func (conf *AppConfig) EncryptSHA256File(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Println(err.Error() + "; in opening specified filepath")
		return nil, err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Println("error in closing the file")
			return
		}
	}(file)

	conf.HmacConf.HmacSigner = sha256.New()
	_, err = io.Copy(conf.HmacConf.HmacSigner, file)
	if err != nil {
		log.Println("error in copying the file into sha writer")
		return nil, err
	}

	eFile := conf.HmacConf.HmacSigner.Sum(nil)

	return eFile, nil
}
