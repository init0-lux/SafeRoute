import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  StyleSheet,
  TouchableOpacity,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  ActivityIndicator,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import { useAuth } from '@/context/AuthContext';
import { ApiError } from '@/services/api';

export default function RegisterScreen() {
  const { register } = useAuth();
  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  const passwordsMatch = password === confirmPassword;
  const canSubmit =
    phone.trim().length > 0 &&
    password.length >= 8 &&
    passwordsMatch &&
    confirmPassword.length > 0 &&
    !isSubmitting;

  const handleRegister = async () => {
    if (!canSubmit) return;

    setErrorMessage('');
    setIsSubmitting(true);

    try {
      await register(phone.trim(), password, email.trim() || undefined);
      router.replace('/home' as never);
    } catch (err) {
      if (err instanceof ApiError) {
        setErrorMessage(err.message);
      } else {
        setErrorMessage('Network error. Please check your connection.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
        keyboardShouldPersistTaps="handled"
      >
        {/* Header */}
        <View style={styles.header}>
          <TouchableOpacity
            onPress={() => router.back()}
            style={styles.backButton}
            activeOpacity={0.7}
          >
            <Ionicons name="arrow-back" size={24} color={Colors.white} />
          </TouchableOpacity>
        </View>

        {/* Branding */}
        <View style={styles.brandContainer}>
          <Text style={styles.title}>Create Account</Text>
          <Text style={styles.subtitle}>join the safety network</Text>
        </View>

        {/* Form */}
        <View style={styles.formContainer}>
          {/* Phone Input */}
          <View style={styles.inputWrapper}>
            <View style={styles.countryCode}>
              <Text style={styles.countryCodeText}>+91</Text>
            </View>
            <TextInput
              style={styles.input}
              placeholder="phone number"
              placeholderTextColor={Colors.gray}
              value={phone}
              onChangeText={setPhone}
              keyboardType="phone-pad"
              autoCapitalize="none"
              autoComplete="tel"
            />
          </View>

          {/* Email Input (Optional) */}
          <View style={styles.inputWrapper}>
            <Ionicons
              name="mail-outline"
              size={18}
              color={Colors.gray}
              style={styles.inputIcon}
            />
            <TextInput
              style={styles.input}
              placeholder="email (optional)"
              placeholderTextColor={Colors.gray}
              value={email}
              onChangeText={setEmail}
              keyboardType="email-address"
              autoCapitalize="none"
              autoComplete="email"
            />
          </View>

          {/* Password Input */}
          <View style={styles.inputWrapper}>
            <Ionicons
              name="lock-closed-outline"
              size={18}
              color={Colors.gray}
              style={styles.inputIcon}
            />
            <TextInput
              style={styles.input}
              placeholder="password"
              placeholderTextColor={Colors.gray}
              value={password}
              onChangeText={setPassword}
              secureTextEntry={!showPassword}
              autoCapitalize="none"
            />
            <TouchableOpacity
              onPress={() => setShowPassword(!showPassword)}
              style={styles.eyeButton}
              hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}
            >
              <Ionicons
                name={showPassword ? 'eye-off-outline' : 'eye-outline'}
                size={20}
                color={Colors.gray}
              />
            </TouchableOpacity>
          </View>

          {/* Confirm Password Input */}
          <View style={styles.inputWrapper}>
            <Ionicons
              name="lock-closed-outline"
              size={18}
              color={Colors.gray}
              style={styles.inputIcon}
            />
            <TextInput
              style={styles.input}
              placeholder="confirm password"
              placeholderTextColor={Colors.gray}
              value={confirmPassword}
              onChangeText={setConfirmPassword}
              secureTextEntry={!showConfirmPassword}
              autoCapitalize="none"
            />
            <TouchableOpacity
              onPress={() => setShowConfirmPassword(!showConfirmPassword)}
              style={styles.eyeButton}
              hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}
            >
              <Ionicons
                name={showConfirmPassword ? 'eye-off-outline' : 'eye-outline'}
                size={20}
                color={Colors.gray}
              />
            </TouchableOpacity>
          </View>

          {/* Password hint */}
          <View style={styles.hintRow}>
            <Ionicons
              name={password.length >= 8 ? 'checkmark-circle' : 'information-circle-outline'}
              size={14}
              color={password.length >= 8 ? Colors.success : Colors.gray}
            />
            <Text
              style={[
                styles.hintText,
                password.length >= 8 && styles.hintTextSuccess,
              ]}
            >
              minimum 8 characters
            </Text>
          </View>

          {/* Password match indicator */}
          {confirmPassword.length > 0 && (
            <View style={styles.hintRow}>
              <Ionicons
                name={passwordsMatch ? 'checkmark-circle' : 'close-circle'}
                size={14}
                color={passwordsMatch ? Colors.success : Colors.red}
              />
              <Text
                style={[
                  styles.hintText,
                  passwordsMatch ? styles.hintTextSuccess : styles.hintTextError,
                ]}
              >
                {passwordsMatch ? 'passwords match' : 'passwords do not match'}
              </Text>
            </View>
          )}

          {/* Error Message */}
          {errorMessage ? (
            <View style={styles.errorContainer}>
              <Ionicons name="alert-circle" size={16} color={Colors.red} />
              <Text style={styles.errorText}>{errorMessage}</Text>
            </View>
          ) : null}

          {/* Register Button */}
          <TouchableOpacity
            style={[styles.registerButton, !canSubmit && styles.registerButtonDisabled]}
            onPress={handleRegister}
            activeOpacity={0.85}
            disabled={!canSubmit}
          >
            {isSubmitting ? (
              <ActivityIndicator size="small" color={Colors.bgDark} />
            ) : (
              <Text style={styles.registerButtonText}>Register</Text>
            )}
          </TouchableOpacity>

          {/* Login Link */}
          <View style={styles.loginRow}>
            <Text style={styles.loginText}>Already have an account? </Text>
            <TouchableOpacity onPress={() => router.back()}>
              <Text style={styles.loginLink}>Login</Text>
            </TouchableOpacity>
          </View>
        </View>

        {/* Legal */}
        <Text style={styles.legalText}>
          By continuing you agree to our{' '}
          <Text style={styles.legalLink}>Terms</Text> &{' '}
          <Text style={styles.legalLink}>Privacy Policy</Text>
        </Text>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.bgDark,
  },
  scrollContent: {
    flexGrow: 1,
    paddingHorizontal: Spacing.xl,
    paddingBottom: Spacing.xxl,
  },
  header: {
    paddingTop: 56,
    paddingBottom: Spacing.lg,
  },
  backButton: {
    width: 40,
    height: 40,
    justifyContent: 'center',
  },
  brandContainer: {
    marginBottom: Spacing.xl,
  },
  title: {
    fontSize: FontSizes.xl + 4,
    fontWeight: '800',
    color: Colors.white,
    marginBottom: Spacing.xs,
  },
  subtitle: {
    fontSize: FontSizes.md,
    color: Colors.grayLight,
    fontWeight: '400',
  },
  formContainer: {
    gap: Spacing.md,
    marginBottom: Spacing.xxl,
  },
  inputWrapper: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.cardBg,
    borderRadius: BorderRadius.md,
    paddingHorizontal: Spacing.md,
    height: 56,
  },
  countryCode: {
    paddingRight: Spacing.md,
    borderRightWidth: 1,
    borderRightColor: Colors.border,
    marginRight: Spacing.md,
    height: '60%',
    justifyContent: 'center',
  },
  countryCodeText: {
    fontSize: FontSizes.md,
    color: Colors.grayLight,
    fontWeight: '500',
  },
  inputIcon: {
    marginRight: Spacing.sm,
  },
  input: {
    flex: 1,
    fontSize: FontSizes.md,
    color: Colors.white,
    height: '100%',
  },
  eyeButton: {
    padding: Spacing.xs,
  },
  hintRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.xs + 2,
    paddingHorizontal: Spacing.xs,
  },
  hintText: {
    fontSize: FontSizes.xs,
    color: Colors.gray,
  },
  hintTextSuccess: {
    color: Colors.success,
  },
  hintTextError: {
    color: Colors.red,
  },
  errorContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm,
    paddingHorizontal: Spacing.xs,
  },
  errorText: {
    fontSize: FontSizes.sm,
    color: Colors.red,
    flex: 1,
  },
  registerButton: {
    backgroundColor: Colors.purple,
    borderRadius: BorderRadius.full,
    height: 56,
    justifyContent: 'center',
    alignItems: 'center',
    marginTop: Spacing.sm,
  },
  registerButtonDisabled: {
    opacity: 0.5,
  },
  registerButtonText: {
    fontSize: FontSizes.md,
    fontWeight: '700',
    color: Colors.bgDark,
  },
  loginRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    marginTop: Spacing.md,
  },
  loginText: {
    fontSize: FontSizes.sm,
    color: Colors.grayLight,
  },
  loginLink: {
    fontSize: FontSizes.sm,
    color: Colors.purple,
    fontWeight: '600',
  },
  legalText: {
    fontSize: FontSizes.xs,
    color: '#666666',
    textAlign: 'center',
    lineHeight: FontSizes.xs * 1.6,
  },
  legalLink: {
    color: Colors.grayLight,
    fontWeight: '500',
  },
});
