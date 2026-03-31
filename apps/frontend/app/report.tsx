import React, { useState } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    TextInput,
    ScrollView,
    Alert,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import { IncidentType } from '@/constants/config';
import IncidentTypeSelector from '@/components/IncidentTypeSelector';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function ReportScreen() {
    const [selectedTypes, setSelectedTypes] = useState<IncidentType[]>([]);
    const [selectorVisible, setSelectorVisible] = useState(false);
    const [description, setDescription] = useState('');

    const handleToggleType = (type: IncidentType) => {
        setSelectedTypes((prev) =>
            prev.includes(type)
                ? prev.filter((t) => t !== type)
                : [...prev, type]
        );
    };

    const handleSubmit = () => {
        if (selectedTypes.length === 0) {
            Alert.alert('Please select at least one incident type');
            return;
        }
        Alert.alert(
            'Report Submitted',
            `Types: ${selectedTypes.join(', ')}\n\nDescription: ${description || 'None'}`,
        );
    };

    return (
        <SafeAreaView style={styles.container}>
            {/* Header */}
            <View style={styles.header}>
                <TouchableOpacity onPress={() => router.back()} activeOpacity={0.7}>
                    <Ionicons name="arrow-back" size={24} color={Colors.white} />
                </TouchableOpacity>
                <Text style={styles.headerTitle}>Report Incident</Text>
                <View style={{ width: 24 }} />
            </View>

            <ScrollView
                style={styles.content}
                contentContainerStyle={styles.contentInner}
                showsVerticalScrollIndicator={false}
            >
                {/* Incident Type Selector */}
                <View style={styles.field}>
                    <Text style={styles.label}>Incident Type</Text>
                    <TouchableOpacity
                        style={styles.selectorButton}
                        onPress={() => setSelectorVisible(true)}
                        activeOpacity={0.7}
                    >
                        <Text
                            style={[
                                styles.selectorText,
                                selectedTypes.length > 0 && styles.selectorTextActive,
                            ]}
                            numberOfLines={2}
                        >
                            {selectedTypes.length > 0
                                ? selectedTypes.join(', ')
                                : 'Select incident type(s)'}
                        </Text>
                        <View style={styles.badge}>
                            {selectedTypes.length > 0 && (
                                <Text style={styles.badgeText}>{selectedTypes.length}</Text>
                            )}
                            <Ionicons name="chevron-down" size={20} color={Colors.grayLight} />
                        </View>
                    </TouchableOpacity>

                    {/* Selected chips */}
                    {selectedTypes.length > 0 && (
                        <View style={styles.chipsContainer}>
                            {selectedTypes.map((type) => (
                                <TouchableOpacity
                                    key={type}
                                    style={styles.chip}
                                    onPress={() => handleToggleType(type)}
                                    activeOpacity={0.7}
                                >
                                    <Text style={styles.chipText}>{type}</Text>
                                    <Ionicons name="close" size={14} color={Colors.purple} />
                                </TouchableOpacity>
                            ))}
                        </View>
                    )}
                </View>

                {/* Description */}
                <View style={styles.field}>
                    <Text style={styles.label}>Description (Optional)</Text>
                    <TextInput
                        style={styles.textArea}
                        placeholder="Describe what happened..."
                        placeholderTextColor={Colors.gray}
                        value={description}
                        onChangeText={setDescription}
                        multiline
                        numberOfLines={5}
                        textAlignVertical="top"
                    />
                </View>

                {/* Evidence Upload Placeholder */}
                <View style={styles.field}>
                    <Text style={styles.label}>Evidence (Optional)</Text>
                    <TouchableOpacity style={styles.uploadArea} activeOpacity={0.7}>
                        <Ionicons name="cloud-upload-outline" size={36} color={Colors.purple} />
                        <Text style={styles.uploadText}>Tap to attach photos or audio</Text>
                        <Text style={styles.uploadSubtext}>Tamper-proof evidence per Section 63(4)</Text>
                    </TouchableOpacity>
                </View>

                {/* Submit Button */}
                <TouchableOpacity
                    style={[
                        styles.submitButton,
                        selectedTypes.length === 0 && styles.submitButtonDisabled,
                    ]}
                    onPress={handleSubmit}
                    activeOpacity={0.85}
                >
                    <Ionicons name="shield-checkmark" size={20} color={Colors.bgDark} />
                    <Text style={styles.submitText}>Submit Report</Text>
                </TouchableOpacity>

                <Text style={styles.disclaimer}>
                    Your report is anonymous and encrypted. Location and timestamp are
                    auto-captured for verification.
                </Text>
            </ScrollView>

            {/* Incident Type Modal */}
            <IncidentTypeSelector
                selectedTypes={selectedTypes}
                onToggleType={handleToggleType}
                visible={selectorVisible}
                onClose={() => setSelectorVisible(false)}
            />
        </SafeAreaView>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: Spacing.lg,
        paddingTop: Spacing.xxl,
        paddingBottom: Spacing.lg,
    },
    headerTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '700',
        color: Colors.white,
    },
    content: {
        flex: 1,
    },
    contentInner: {
        paddingHorizontal: Spacing.lg,
        paddingBottom: Spacing.xxl,
    },
    field: {
        marginBottom: Spacing.xl,
    },
    label: {
        fontSize: FontSizes.sm,
        fontWeight: '600',
        color: Colors.grayLight,
        textTransform: 'uppercase',
        letterSpacing: 1,
        marginBottom: Spacing.sm,
    },
    selectorButton: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        backgroundColor: Colors.cardBg,
        borderRadius: BorderRadius.md,
        paddingVertical: Spacing.md,
        paddingHorizontal: Spacing.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    selectorText: {
        fontSize: FontSizes.md,
        color: Colors.gray,
        flex: 1,
        marginRight: Spacing.sm,
    },
    selectorTextActive: {
        color: Colors.white,
    },
    badge: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 6,
    },
    badgeText: {
        fontSize: FontSizes.xs,
        fontWeight: '700',
        color: Colors.bgDark,
        backgroundColor: Colors.purple,
        borderRadius: BorderRadius.full,
        width: 20,
        height: 20,
        textAlign: 'center',
        lineHeight: 20,
        overflow: 'hidden',
    },
    chipsContainer: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: Spacing.sm,
        marginTop: Spacing.md,
    },
    chip: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: 'rgba(201, 160, 220, 0.15)',
        borderRadius: BorderRadius.full,
        paddingVertical: Spacing.xs,
        paddingLeft: Spacing.md,
        paddingRight: Spacing.sm,
        gap: 6,
        borderWidth: 1,
        borderColor: 'rgba(201, 160, 220, 0.3)',
    },
    chipText: {
        fontSize: FontSizes.xs,
        color: Colors.purpleLight,
        fontWeight: '500',
    },
    textArea: {
        backgroundColor: Colors.cardBg,
        borderRadius: BorderRadius.md,
        paddingHorizontal: Spacing.md,
        paddingVertical: Spacing.md,
        fontSize: FontSizes.md,
        color: Colors.white,
        minHeight: 120,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    uploadArea: {
        backgroundColor: Colors.cardBg,
        borderRadius: BorderRadius.md,
        paddingVertical: Spacing.xl,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: Colors.border,
        borderStyle: 'dashed',
    },
    uploadText: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        marginTop: Spacing.sm,
        fontWeight: '500',
    },
    uploadSubtext: {
        fontSize: FontSizes.xs,
        color: Colors.gray,
        marginTop: Spacing.xs,
    },
    submitButton: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: Colors.purple,
        borderRadius: BorderRadius.full,
        paddingVertical: Spacing.md + 2,
        gap: Spacing.sm,
        marginTop: Spacing.md,
    },
    submitButtonDisabled: {
        opacity: 0.5,
    },
    submitText: {
        fontSize: FontSizes.md,
        fontWeight: '700',
        color: Colors.bgDark,
    },
    disclaimer: {
        fontSize: FontSizes.xs,
        color: Colors.gray,
        textAlign: 'center',
        marginTop: Spacing.lg,
        lineHeight: FontSizes.xs * 1.6,
    },
});
