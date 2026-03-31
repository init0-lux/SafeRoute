import React from 'react';
import {
    View,
    Text,
    TouchableOpacity,
    StyleSheet,
    ScrollView,
    Modal,
    Pressable,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import { INCIDENT_TYPES, IncidentType } from '@/constants/config';

interface IncidentTypeSelectorProps {
    selectedTypes: IncidentType[];
    onToggleType: (type: IncidentType) => void;
    visible: boolean;
    onClose: () => void;
}

export default function IncidentTypeSelector({
    selectedTypes,
    onToggleType,
    visible,
    onClose,
}: IncidentTypeSelectorProps) {
    return (
        <Modal
            visible={visible}
            transparent
            animationType="slide"
            onRequestClose={onClose}
        >
            <Pressable style={styles.overlay} onPress={onClose}>
                <Pressable style={styles.modal} onPress={() => { }}>
                    <View style={styles.header}>
                        <Text style={styles.headerTitle}>Select Incident Type(s)</Text>
                        <TouchableOpacity onPress={onClose} activeOpacity={0.7}>
                            <Ionicons name="close" size={24} color={Colors.white} />
                        </TouchableOpacity>
                    </View>

                    <Text style={styles.subtext}>You can select multiple types</Text>

                    <ScrollView
                        style={styles.list}
                        showsVerticalScrollIndicator={false}
                        contentContainerStyle={styles.listContent}
                    >
                        {INCIDENT_TYPES.map((type) => {
                            const isSelected = selectedTypes.includes(type);
                            return (
                                <TouchableOpacity
                                    key={type}
                                    style={[styles.item, isSelected && styles.itemSelected]}
                                    onPress={() => onToggleType(type)}
                                    activeOpacity={0.7}
                                >
                                    <View style={[styles.checkbox, isSelected && styles.checkboxSelected]}>
                                        {isSelected && (
                                            <Ionicons name="checkmark" size={14} color={Colors.bgDark} />
                                        )}
                                    </View>
                                    <Text style={[styles.itemText, isSelected && styles.itemTextSelected]}>
                                        {type}
                                    </Text>
                                </TouchableOpacity>
                            );
                        })}
                    </ScrollView>

                    <TouchableOpacity style={styles.doneButton} onPress={onClose} activeOpacity={0.85}>
                        <Text style={styles.doneText}>
                            Done {selectedTypes.length > 0 ? `(${selectedTypes.length})` : ''}
                        </Text>
                    </TouchableOpacity>
                </Pressable>
            </Pressable>
        </Modal>
    );
}

const styles = StyleSheet.create({
    overlay: {
        flex: 1,
        backgroundColor: Colors.overlay,
        justifyContent: 'flex-end',
    },
    modal: {
        backgroundColor: Colors.cardBg,
        borderTopLeftRadius: BorderRadius.xl,
        borderTopRightRadius: BorderRadius.xl,
        paddingTop: Spacing.lg,
        paddingHorizontal: Spacing.lg,
        paddingBottom: Spacing.xxl,
        maxHeight: '80%',
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: Spacing.xs,
    },
    headerTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '700',
        color: Colors.white,
    },
    subtext: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        marginBottom: Spacing.lg,
    },
    list: {
        flexGrow: 0,
    },
    listContent: {
        gap: Spacing.sm,
        paddingBottom: Spacing.md,
    },
    item: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.cardBgLight,
        borderRadius: BorderRadius.md,
        paddingVertical: Spacing.md,
        paddingHorizontal: Spacing.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    itemSelected: {
        borderColor: Colors.purple,
        backgroundColor: 'rgba(201, 160, 220, 0.1)',
    },
    checkbox: {
        width: 22,
        height: 22,
        borderRadius: BorderRadius.sm,
        borderWidth: 2,
        borderColor: Colors.gray,
        justifyContent: 'center',
        alignItems: 'center',
        marginRight: Spacing.md,
    },
    checkboxSelected: {
        backgroundColor: Colors.purple,
        borderColor: Colors.purple,
    },
    itemText: {
        fontSize: FontSizes.md,
        color: Colors.white,
        flex: 1,
    },
    itemTextSelected: {
        color: Colors.purpleLight,
        fontWeight: '600',
    },
    doneButton: {
        backgroundColor: Colors.purple,
        borderRadius: BorderRadius.full,
        paddingVertical: Spacing.md,
        alignItems: 'center',
        marginTop: Spacing.md,
    },
    doneText: {
        fontSize: FontSizes.md,
        fontWeight: '700',
        color: Colors.bgDark,
    },
});
