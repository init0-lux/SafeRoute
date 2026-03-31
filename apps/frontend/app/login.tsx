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
  Alert,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import { useAuth } from '@/context/AuthContext';
import { ApiError } from '@/services/api';

export default function LoginScreen() {
  const { login } = useAuth();
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  const canSubmit = phone.trim().length > 0 && password.length >= 8 && !isSubmitting;

  const handleLogin = async () => {
    if (!canSubmit) return;

    setErrorMessage('');
    setIsSubmitting(true);

    try {
      await login(phone.trim(), password);
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
        {/* Logo / Branding */}
        <View style={styles.brandContainer}>
          <View style={styles.logoRow}>
            <Ionicons name="shield-checkmark" size={32} color={Colors.purple} />
            <Text style={styles.logoText}>SafeRoute</Text>
          </View>
          <Text style={styles.subtitle}>your safety companion</Text>
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
              autoComplete="password"
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

          {/* Error Message */}
          {errorMessage ? (
            <View style={styles.errorContainer}>
              <Ionicons name="alert-circle" size={16} color={Colors.red} />
              <Text style={styles.errorText}>{errorMessage}</Text>
            </View>
          ) : null}

          {/* Login Button */}
          <TouchableOpacity
            style={[styles.loginButton, !canSubmit && styles.loginButtonDisabled]}
            onPress={handleLogin}
            activeOpacity={0.85}
            disabled={!canSubmit}
          >
            {isSubmitting ? (
              <ActivityIndicator size="small" color={Colors.bgDark} />
            ) : (
              <Text style={styles.loginButtonText}>Login</Text>
            )}
          </TouchableOpacity>

          {/* Register Link */}
          <View style={styles.registerRow}>
            <Text style={styles.registerText}>Don't have an account? </Text>
            <TouchableOpacity onPress={() => router.push('/register' as never)}>
              <Text style={styles.registerLink}>Register</Text>
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
    justifyContent: 'center',
    paddingHorizontal: Spacing.xl,
    paddingVertical: Spacing.xxl,
  },
  brandContainer: {
    alignItems: 'center',
    marginBottom: Spacing.xxl + 16,
  },
  logoRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm,
    marginBottom: Spacing.sm,
  },
  logoText: {
    fontSize: FontSizes.xxl,
    fontWeight: '800',
    color: Colors.white,
    letterSpacing: -0.5,
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
  loginButton: {
    backgroundColor: Colors.purple,
    borderRadius: BorderRadius.full,
    height: 56,
    justifyContent: 'center',
    alignItems: 'center',
    marginTop: Spacing.sm,
  },
  loginButtonDisabled: {
    opacity: 0.5,
  },
  loginButtonText: {
    fontSize: FontSizes.md,
    fontWeight: '700',
    color: Colors.bgDark,
  },
  registerRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    marginTop: Spacing.md,
  },
  registerText: {
    fontSize: FontSizes.sm,
    color: Colors.grayLight,
  },
  registerLink: {
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
